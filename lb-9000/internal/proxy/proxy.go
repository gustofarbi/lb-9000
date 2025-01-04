package proxy

import (
	"lb-9000/lb-9000/internal/pool"
	"log/slog"
	"net/http"
	"net/http/httputil"
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

	slog.Info("listening on", "port", port)

	server := http.Server{
		Addr:    ":" + port,
		Handler: proxy,
		// todo config
		ReadHeaderTimeout: 30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("error listening for connections", "err", err)
	}
}
