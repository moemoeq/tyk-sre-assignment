package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	disco "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetKubernetesVersion(t *testing.T) {
	// Test Case 1: Success
	okClientset := fake.NewSimpleClientset()
	okClientset.Discovery().(*disco.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: "1.25.0-fake"}

	okVer, err := GetKubernetesVersion(okClientset)
	assert.NoError(t, err)
	assert.Equal(t, "1.25.0-fake", okVer)

	badClientset := fake.NewSimpleClientset()
	badClientset.Discovery().(*disco.FakeDiscovery).FakedServerVersion = &version.Info{}

	badVer, err := GetKubernetesVersion(badClientset)
	assert.NoError(t, err)
	assert.Equal(t, "", badVer)
}

func TestListDeployments(t *testing.T) {
	ctx := context.Background()

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-deploy",
			Namespace: "fake-namespace",
		},
	}

	client := &Client{
		Clientset: fake.NewSimpleClientset(deploy),
	}

	d, err := client.ListDeployments(ctx, "fake-namespace")
	assert.NoError(t, err)
	assert.Len(t, d, 1)
	assert.Equal(t, "fake-deploy", d[0].Name)
}
func TestCheckConnectivity(t *testing.T) {
	ctx := context.Background()

	client := &Client{
		Clientset: fake.NewSimpleClientset(),
	}
	client.Clientset.Discovery().(*disco.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: "1.25.0-fake"}
	status := client.CheckConnectivity(ctx)

	// Because we use fake client, we can't check
	// assert.True(t, status.Status)
	// Reachability is not checked in fake client
	// assert.True(t, status.Reachability)
	assert.True(t, status.Discovery)
	assert.Equal(t, "1.25.0-fake", status.Version)
	assert.Empty(t, status.Error)
}
