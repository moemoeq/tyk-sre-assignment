package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
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

// if ns is empty, it returns all across all namespaces.
func (c *Client) ListNetworkPolicies(ctx context.Context, namespace string, opts metav1.ListOptions) ([]networkingv1.NetworkPolicy, error) {
	pols, err := c.Clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return pols.Items, nil
}

func (c *Client) CreateNetworkPolicy(ctx context.Context, policy *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
	return c.Clientset.NetworkingV1().NetworkPolicies(policy.Namespace).Create(ctx, policy, metav1.CreateOptions{})
}

// delete by name and namespace.
func (c *Client) DeleteNetworkPolicy(ctx context.Context, namespace, name string) error {
	return c.Clientset.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// Search by UID.
// It searches in the specified namespace, or all namespaces if "namespace" is empty.
func (c *Client) GetNetworkPolicyByUID(ctx context.Context, namespace, uid string) (*networkingv1.NetworkPolicy, error) {
	list, err := c.ListNetworkPolicies(ctx, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, policy := range list {
		if string(policy.UID) == uid {
			return &policy, nil
		}
	}

	return nil, fmt.Errorf("network policy with UID %s not found", uid)
}

// Delete a NetworkPolicy by UID.
func (c *Client) DeleteNetworkPolicyByUID(ctx context.Context, namespace, uid string) error {
	policy, err := c.GetNetworkPolicyByUID(ctx, namespace, uid)
	if err != nil {
		return err
	}
	return c.DeleteNetworkPolicy(ctx, policy.Namespace, policy.Name)
}
