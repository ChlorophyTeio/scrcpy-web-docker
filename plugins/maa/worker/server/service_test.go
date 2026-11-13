// Modified for the optional-plugins distribution (2026). See NOTICE.
package server

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func logTestEntry(message string) *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return &logrus.Entry{Logger: logger, Time: time.Date(2026, 11, 13, 12, 34, 56, 0, time.FixedZone("test", 8*60*60)), Level: logrus.WarnLevel, Message: message}
}
func TestLogStreamBoundedHistoryAndUTF8(t *testing.T) {
	stream := newLogStream()
	for i := 0; i < logHistoryLimit+20; i++ {
		if err := stream.Fire(logTestEntry("same message")); err != nil {
			t.Fatal(err)
		}
	}
	if len(stream.history) != logHistoryLimit || stream.history[0].ID != 21 || stream.history[len(stream.history)-1].ID != 320 {
		t.Fatal("history lost event ordering")
	}
	if stream.history[0].Message != "same message" || stream.history[0].Level != "warning" || stream.history[0].Time != "2026-11-13T12:34:56+08:00" {
		t.Fatal("incorrect structured metadata")
	}
	if newLogStream().id == stream.id || stream.id == "" {
		t.Fatal("streams must distinguish server restarts")
	}
	if err := stream.Fire(logTestEntry(strings.Repeat("中文🙂", 4000))); err != nil {
		t.Fatal(err)
	}
	last := stream.history[len(stream.history)-1]
	if !last.Truncated || len(last.Raw) > logTextLimit || len(last.Message) > logTextLimit || !utf8.ValidString(last.Raw) || !utf8.ValidString(last.Message) {
		t.Fatal("long messages must be bounded valid UTF-8")
	}
}
func TestLogStreamConcurrentWritersAndSlowConsumer(t *testing.T) {
	stream := newLogStream()
	ch := make(chan logRecord, 1)
	stream.clients[ch] = struct{}{}
	var group sync.WaitGroup
	for i := 0; i < 5; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 80; j++ {
				if err := stream.Fire(logTestEntry("repeat")); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	done := make(chan struct{})
	go func() { group.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("slow client blocked logging")
	}
	if stream.sequence != 400 {
		t.Fatal("concurrent logs lost")
	}
	for i, row := range stream.history {
		if row.ID != uint64(101+i) {
			t.Fatal("history sequence out of order")
		}
	}
}
func TestLogWebSocketHistoryAndLiveRemainBackwardCompatible(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stream := newLogStream()
	if err := stream.Fire(logTestEntry("开始任务：信用收支")); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/ws", stream.serve)
	server := httptest.NewServer(router)
	defer server.Close()
	connect := func() *websocket.Conn {
		t.Helper()
		c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws", nil)
		if err != nil {
			t.Fatal(err)
		}
		c.SetReadDeadline(time.Now().Add(5 * time.Second))
		return c
	}
	type frame struct {
		History []string    `json:"history"`
		Entries []logRecord `json:"entries"`
		Msg     string      `json:"msg"`
		Entry   logRecord   `json:"entry"`
		Stream  string      `json:"stream_id"`
	}
	read := func(c *websocket.Conn) frame {
		t.Helper()
		var f frame
		if err := c.ReadJSON(&f); err != nil {
			t.Fatal(err)
		}
		return f
	}
	c := connect()
	defer c.Close()
	initial := read(c)
	if initial.Stream != stream.id || len(initial.History) != 1 || len(initial.Entries) != 1 || initial.Entries[0].ID != 1 || !strings.Contains(initial.History[0], "信用收支") {
		t.Fatal("history metadata must align with legacy raw logs")
	}
	if err := stream.Fire(logTestEntry("任务未能确认")); err != nil {
		t.Fatal(err)
	}
	live := read(c)
	if live.Stream != initial.Stream || live.Entry.ID != 2 || live.Entry.Message != "任务未能确认" || !strings.Contains(live.Msg, "任务未能确认") {
		t.Fatal("live event metadata must align with legacy msg")
	}
	c2 := connect()
	defer c2.Close()
	replay := read(c2)
	if len(replay.Entries) != 2 || replay.Entries[1].ID != live.Entry.ID || replay.Stream != live.Stream {
		t.Fatal("reconnect must replay stable event IDs")
	}
	encoded, _ := json.Marshal(replay.Entries)
	if strings.Contains(string(encoded), `"Raw"`) {
		t.Fatal("raw history must not be duplicated inside metadata")
	}
}
