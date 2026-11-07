package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type logStream struct {
	mu      sync.Mutex
	history []string
	clients map[chan string]struct{}
}

func (h *logStream) Levels() []logrus.Level { return logrus.AllLevels }
func (h *logStream) Fire(e *logrus.Entry) error {
	line, err := e.String()
	if err != nil {
		return err
	}
	if len(line) > 8192 {
		line = line[:8192] + "…"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history = append(h.history, line)
	if len(h.history) > 300 {
		h.history = append([]string{}, h.history[len(h.history)-300:]...)
	}
	for ch := range h.clients {
		select {
		case ch <- line:
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
	ch := make(chan string, 32)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	history := append([]string{}, h.history...)
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
	if conn.WriteJSON(gin.H{"history": history}) != nil {
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
			if conn.WriteJSON(gin.H{"msg": line}) != nil {
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
	stream := &logStream{clients: map[chan string]struct{}{}}
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
