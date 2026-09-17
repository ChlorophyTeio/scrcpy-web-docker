package wskeepalive

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestStartClosesConnectionWithoutPong(t *testing.T) {
	upgrader := websocket.Upgrader{}
	closed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer close(closed)
		defer conn.Close()
		stop := Start(conn, Options{PingInterval: 10 * time.Millisecond, PongTimeout: 40 * time.Millisecond, WriteTimeout: 10 * time.Millisecond})
		defer stop()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("connection remained open without pong processing")
	}
}
