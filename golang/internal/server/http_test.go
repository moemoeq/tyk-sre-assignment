package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/moemoeq/tyk-sre-app/internal/k8s"
	"github.com/moemoeq/tyk-sre-app/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/version"
	disco "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h := NewHandler()
	h.Health(rec, req)
	res := rec.Result()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	defer func(Body io.ReadCloser) {
		assert.NoError(t, Body.Close())
	}(res.Body)
	resp, err := io.ReadAll(res.Body)

	assert.NoError(t, err)
	assert.Equal(t, "ok", string(resp))
}

func TestMetricsHandler(t *testing.T) {
	// Making Fake Mock k8s Clientset
	okClientset := fake.NewSimpleClientset()
	okClientset.Discovery().(*disco.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: "1.25.0-fake"}

	metrics.Init(context.TODO(), prometheus.DefaultRegisterer, okClientset)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	h := NewHandler()
	h.Metrics(rec, req)
	res := rec.Result()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	defer func(Body io.ReadCloser) {
		assert.NoError(t, Body.Close())
	}(res.Body)
	resp, err := io.ReadAll(res.Body)

	assert.NoError(t, err)
	output := string(resp)

	// Verify Go version in go_info
	goVersion := runtime.Version()
	assert.Contains(t, output, fmt.Sprintf("go_info{version=\"%s\"} 1", goVersion))

	okVer, err := k8s.GetKubernetesVersion(okClientset)
	assert.NoError(t, err)
	assert.Contains(t, output, fmt.Sprintf("k8s_api_server_version{version=\"%s\"} 1", okVer))
}
