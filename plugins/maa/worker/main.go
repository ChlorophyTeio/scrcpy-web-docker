package main

import (
	"context"
	"log"
	"maa-server/server"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	address := os.Getenv("MAA_LISTEN_ADDRESS")
	if address == "" {
		address = ":8080"
	}
	srv := &http.Server{Addr: address, Handler: server.NewHandler(), ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
		srv.Shutdown(shutdown)
	}()
	log.Printf("MAA plugin listening on %s", address)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
