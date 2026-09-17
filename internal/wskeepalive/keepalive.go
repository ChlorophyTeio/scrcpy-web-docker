package wskeepalive

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Options struct {
	PingInterval time.Duration
	PongTimeout  time.Duration
	WriteTimeout time.Duration
}

var DefaultOptions = Options{
	PingInterval: 15 * time.Second,
	PongTimeout:  45 * time.Second,
	WriteTimeout: 15 * time.Second,
}

func Start(conn *websocket.Conn, options Options) func() {
	options = options.withDefaults()
	_ = conn.SetReadDeadline(time.Now().Add(options.PongTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(options.PongTimeout))
	})

	stop := make(chan struct{})
	var once sync.Once
	go func() {
		ticker := time.NewTicker(options.PingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(options.WriteTimeout)); err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()

	return func() {
		once.Do(func() { close(stop) })
	}
}

func (o Options) withDefaults() Options {
	if o.PingInterval <= 0 {
		o.PingInterval = DefaultOptions.PingInterval
	}
	if o.PongTimeout <= 0 {
		o.PongTimeout = DefaultOptions.PongTimeout
	}
	if o.WriteTimeout <= 0 {
		o.WriteTimeout = DefaultOptions.WriteTimeout
	}
	return o
}
