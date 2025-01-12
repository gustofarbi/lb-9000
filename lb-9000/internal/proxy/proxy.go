package proxy

import (
	"context"
	"lb-9000/lb-9000/internal/pool"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Start(pool *pool.Pool, port string) {
	pool.Init()

	proxy := &httputil.ReverseProxy{
		Director:       pool.Director,
		ModifyResponse: pool.ModifyResponse,
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {})
		healthServer := http.Server{
			Addr:              ":8081",
			Handler:           mux,
			ReadHeaderTimeout: 30 * time.Second,
		}

		if err := healthServer.ListenAndServe(); err != nil {
			slog.Error("error serving health requests", "err", err)
		}
	}()

	server := http.Server{
		Addr:    ":" + port,
		Handler: proxy,
		// todo config
		ReadHeaderTimeout: 30 * time.Second,
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopCh
		slog.Info("shutting down")
		if err := server.Shutdown(context.Background()); err != nil {
			slog.Error("error shutting down server", "err", err)
		}
	}()

	slog.Info("listening on", "port", port)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("error listening for connections", "err", err)
	}
}
