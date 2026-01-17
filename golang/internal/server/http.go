package server

import (
	"net/http"
	"time"

	v1 "github.com/moemoeq/tyk-sre-app/internal/api/v1"
)

func New(addr string, apiV1 *v1.API) *http.Server {
	mux := http.NewServeMux()

	h := NewHandler()
	mux.HandleFunc("/healthz", h.Health)

	apiV1Mux := http.NewServeMux()
	apiV1.Register(apiV1Mux)

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiV1Mux))

	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
