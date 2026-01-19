package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes clientset.
type Client struct {
	Clientset kubernetes.Interface
}

// NewClient creates a new Kubernetes client based on the provided kubeconfig path
// or in-cluster config if path is empty.
func NewClient(kubeconfig string) (*Client, error) {
	kConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kConfig)
	if err != nil {
		return nil, err
	}

	return &Client{
		Clientset: clientset,
	}, nil
}

func GetKubernetesVersion(clientset kubernetes.Interface) (string, error) {
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.String(), nil
}

// ReachabilityStatus represents the detailed status of the Kubernetes API server connectivity.
type ReachabilityStatus struct {
	Status       bool   `json:"status"`
	Reachability bool   `json:"reachability"`
	Discovery    bool   `json:"discovery"`
	Error        string `json:"error,omitempty"`
	Version      string `json:"version,omitempty"`
}

// Check Connectivity between the Kubernetes API server
func (c *Client) CheckConnectivity(ctx context.Context) ReachabilityStatus {
	status := ReachabilityStatus{}

	// check api reachability
	if rest := c.Clientset.Discovery().RESTClient(); rest != nil {
		res := rest.Get().AbsPath("/healthz").Do(ctx)
		if err := res.Error(); err != nil {
			status.Error = fmt.Sprintf("reachability error: %v", err)
		} else {
			status.Reachability = true
		}
	}

	// check api discovery working
	if version, err := GetKubernetesVersion(c.Clientset); err != nil {
		if status.Error != "" {
			status.Error += "; "
		}
		status.Error += fmt.Sprintf("discovery error: %v", err)
	} else {
		status.Discovery = true
		status.Version = version
	}

	status.Status = status.Reachability && status.Discovery

	return status
}

// Get List Deployments leave empty to get all
func (c *Client) ListDeployments(ctx context.Context, namespace string, opts metav1.ListOptions) ([]appsv1.Deployment, error) {
	deps, err := c.Clientset.AppsV1().Deployments(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return deps.Items, nil
}
