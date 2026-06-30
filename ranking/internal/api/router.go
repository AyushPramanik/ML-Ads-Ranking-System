package api

import (
	"log/slog"
	"net/http"
)

// NewRouter wires the routes and wraps them with the middleware stack. The
// middleware order is: request ID -> logging -> panic recovery -> handler.
func NewRouter(h *Handlers, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /ads", h.Ads)
	mux.HandleFunc("POST /rank", h.Rank)
	mux.HandleFunc("POST /features", h.Features)

	return chain(mux,
		RequestID,
		RequestLogger(logger),
		Recover(logger),
	)
}
