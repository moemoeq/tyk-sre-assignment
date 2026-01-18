package server

import (
	"context"
	"net/http"
	"time"

	v1 "github.com/moemoeq/tyk-sre-app/internal/api/v1"
	"github.com/moemoeq/tyk-sre-app/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func New(ctx context.Context, addr string, apiV1 *v1.API) *http.Server {
	metrics.Init(ctx, prometheus.DefaultRegisterer, apiV1.K8sClient.Clientset)
	mux := http.NewServeMux()

	h := NewHandler()
	mux.HandleFunc("/healthz", h.Health)
	mux.HandleFunc("/metrics", h.Metrics)

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
