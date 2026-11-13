package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const logHistoryLimit = 300
const logTextLimit = 8192

type logRecord struct {
	ID        uint64 `json:"id"`
	Time      string `json:"time"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Truncated bool   `json:"truncated,omitempty"`
	Raw       string `json:"-"`
}
type logStream struct {
	mu       sync.Mutex
	id       string
	sequence uint64
	history  []logRecord
	clients  map[chan logRecord]struct{}
}

func newLogStream() *logStream {
	var id [12]byte
	name := ""
	if _, err := rand.Read(id[:]); err == nil {
		name = hex.EncodeToString(id[:])
	} else {
		name = fmt.Sprint(time.Now().UnixNano())
	}
	return &logStream{id: name, clients: map[chan logRecord]struct{}{}}
}
func boundedLogText(value string) (string, bool) {
	value = strings.ToValidUTF8(value, "�")
	if len(value) <= logTextLimit {
		return value, false
	}
	value = value[:logTextLimit-len("…")]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value + "…", true
}
func (h *logStream) Levels() []logrus.Level { return logrus.AllLevels }
func (h *logStream) Fire(e *logrus.Entry) error {
	raw, err := e.String()
	if err != nil {
		return err
	}
	raw, rawCut := boundedLogText(raw)
	message, messageCut := boundedLogText(e.Message)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sequence++
	record := logRecord{ID: h.sequence, Time: e.Time.Format(time.RFC3339Nano), Level: e.Level.String(), Message: message, Truncated: rawCut || messageCut, Raw: raw}
	h.history = append(h.history, record)
	if len(h.history) > logHistoryLimit {
		h.history = append([]logRecord{}, h.history[len(h.history)-logHistoryLimit:]...)
	}
	for ch := range h.clients {
		select {
		case ch <- record:
		default:
		}
	}
	return nil
}
func (h *logStream) serve(c *gin.Context) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }} // Authentication belongs to the private-network plugin gateway.
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	ch := make(chan logRecord, 128)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	entries := append([]logRecord{}, h.history...)
	history := make([]string, len(entries))
	for i, entry := range entries {
		history[i] = entry.Raw
	}
	h.mu.Unlock()
	defer func() { h.mu.Lock(); delete(h.clients, ch); h.mu.Unlock() }()
	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(75 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(75 * time.Second)) })
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, e := conn.ReadMessage(); e != nil {
				return
			}
		}
	}()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if conn.WriteJSON(gin.H{"history": history, "entries": entries, "stream_id": h.id}) != nil {
		return
	}
	tick := time.NewTicker(25 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-done:
			return
		case line := <-ch:
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if conn.WriteJSON(gin.H{"msg": line.Raw, "entry": line, "stream_id": h.id}) != nil {
				return
			}
		case <-tick.C:
			if conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)) != nil {
				return
			}
		}
	}
}
func NewHandler() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		if shuttingDown.Load() {
			c.AbortWithStatusJSON(503, gin.H{"error": "plugin shutting down"})
			return
		}
		c.Next()
	})
	stream := newLogStream()
	logrus.AddHook(stream)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"id": "maa", "api_version": 1}) })
	r.GET("/ws", stream.serve)
	api := r.Group("/api")
	InitGUI(api)
	api.GET("/GetRunningTask", func(c *gin.Context) { gui.Lock(); defer gui.Unlock(); c.JSON(200, gui.state) })
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(404, gin.H{"error": "plugin endpoint not found"})
			return
		}
		c.JSON(200, gin.H{"service": "MAA plugin", "webui": "Open the MAA plugin in ScrcpyCat"})
	})
	return r
}

var shuttingDown atomic.Bool

func Shutdown(ctx context.Context) {
	shuttingDown.Store(true)
	gui.Lock()
	cancel := gui.cancel
	gui.Unlock()
	if cancel != nil {
		cancel()
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		gui.Lock()
		running := gui.state.Running
		gui.Unlock()
		if !running {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
