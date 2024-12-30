package proxy

import (
	"lb-9000/lb-9000/internal/pool"
	"log/slog"
	"net/http"
	"net/http/httputil"
)

func Start(pool *pool.Pool, port string) {
	proxy := httputil.ReverseProxy{
		Director:       pool.Director,
		ModifyResponse: pool.ModifyResponse,
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {})
		if err := http.ListenAndServe(":8081", mux); err != nil {
			slog.Error("error serving health requests", "err", err)
		}
	}()

	slog.Info("listening on", "port", port)
	if err := http.ListenAndServe(":"+port, &proxy); err != nil {
		slog.Error("error listening", "err", err)
	}
}
