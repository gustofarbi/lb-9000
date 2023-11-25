package proxy

import (
	"lb-9000/internal/pool"
	"log/slog"
	"net/http"
	"net/http/httputil"
)

func Start(podPool *pool.Pool, port string) {
	proxy := httputil.ReverseProxy{
		Director:       podPool.Director,
		ModifyResponse: podPool.ModifyResponse,
	}

	slog.Info("listening on", "port", port)
	if err := http.ListenAndServe(":"+port, &proxy); err != nil {
		slog.Error("error listening", "err", err)
	}
}
