package metrics

import (
	"context"
	"fmt"

	"github.com/moemoeq/tyk-sre-app/internal/k8s"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
)

const (
	MetricK8sAPIServerVersion = "k8s_api_server_version"
)

type Metrics struct{}

type k8sCollector struct {
	clientset kubernetes.Interface
}

func (c *k8sCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *k8sCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectVersion(ch)
}

func (c *k8sCollector) collectVersion(ch chan<- prometheus.Metric) {
	version, _ := k8s.GetKubernetesVersion(c.clientset)
	if version == "" {
		// TODO: to find better way to hadle this.
		version = "unknown"
		fmt.Println("failed to get kubernetes version")
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(MetricK8sAPIServerVersion, "Kubernetes API server version.", []string{"version"}, nil),
		prometheus.GaugeValue, 1, version,
	)
}

// register prometheus metrics.
func Init(ctx context.Context, reg prometheus.Registerer, clientset kubernetes.Interface) *Metrics {
	reg.MustRegister(&k8sCollector{clientset: clientset})
	return &Metrics{}
}
