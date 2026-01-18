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
