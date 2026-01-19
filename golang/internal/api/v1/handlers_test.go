package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moemoeq/tyk-sre-app/internal/k8s"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	disco "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetDeployments_Summary(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
		},
	})
	kClient := &k8s.Client{Clientset: fakeClientset}
	api := &API{K8sClient: kClient}

	req, _ := http.NewRequest("GET", "/deployments", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.getDeployments)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var deps []EnrichedDeployment
	err := json.Unmarshal(rr.Body.Bytes(), &deps)
	assert.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "test-deploy", deps[0].Name)
	assert.Nil(t, deps[0].Spec)
}

func TestGetDeployments_Detailed(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
		},
	})
	kClient := &k8s.Client{Clientset: fakeClientset}
	api := &API{K8sClient: kClient}

	req, _ := http.NewRequest("GET", "/deployments?detailed=true", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.getDeployments)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var deps []EnrichedDeployment
	err := json.Unmarshal(rr.Body.Bytes(), &deps)
	assert.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "test-deploy", deps[0].Name)
	assert.NotNil(t, deps[0].Spec)
}

func TestGetDeployments_LabelFilter(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-a",
				Namespace: "default",
				Labels:    map[string]string{"app": "a"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-b",
				Namespace: "default",
				Labels:    map[string]string{"app": "b"},
			},
		},
	)
	kClient := &k8s.Client{Clientset: fakeClientset}
	api := &API{K8sClient: kClient}

	req, _ := http.NewRequest("GET", "/deployments?labelSelector=app=a", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.getDeployments)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var deps []EnrichedDeployment
	err := json.Unmarshal(rr.Body.Bytes(), &deps)
	assert.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, "deploy-a", deps[0].Name)
}

func TestCheckK8sHealth(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	// Explicitly set empty version to trigger discovery failure
	fakeClientset.Discovery().(*disco.FakeDiscovery).FakedServerVersion = &version.Info{}

	kClient := &k8s.Client{Clientset: fakeClientset}
	api := &API{K8sClient: kClient}

	req, _ := http.NewRequest("GET", "/reachability", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.checkK8sReachability)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

	var resp k8s.ReachabilityStatus
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
}
