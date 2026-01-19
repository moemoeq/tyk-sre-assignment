package metrics

import (
	"context"

	"github.com/moemoeq/tyk-sre-app/internal/k8s"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricK8sAPIServerVersion          = "k8s_api_server_version"
	MetricK8sAPIServerReachable        = "k8s_api_server_reachable"
	MetricK8sAPIServerDiscoverySuccess = "k8s_api_server_discovery_success"
)

type Metrics struct{}

type k8sCollector struct {
	client *k8s.Client
}

func (c *k8sCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *k8sCollector) Collect(ch chan<- prometheus.Metric) {
	status := c.client.CheckConnectivity(context.Background())

	// 1. Version Metric
	version := status.Version
	if version == "" {
		version = "unknown"
	}
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(MetricK8sAPIServerVersion, "Kubernetes API server version.", []string{"version"}, nil),
		prometheus.GaugeValue, 1, version,
	)

	// 2. Reachability Metric
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(MetricK8sAPIServerReachable, "Kubernetes API server reachability status.", nil, nil),
		prometheus.GaugeValue, BoolToFloat(status.Reachability),
	)

	// 3. Discovery Metric
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(MetricK8sAPIServerDiscoverySuccess, "Kubernetes API server discovery success status.", nil, nil),
		prometheus.GaugeValue, BoolToFloat(status.Discovery),
	)
}

func BoolToFloat(b bool) float64 {
	return map[bool]float64{
		true:  1.0,
		false: 0.0,
	}[b]
}

// register prometheus metrics.
func Init(ctx context.Context, reg prometheus.Registerer, client *k8s.Client) *Metrics {
	reg.MustRegister(&k8sCollector{client: client})
	return &Metrics{}
}
