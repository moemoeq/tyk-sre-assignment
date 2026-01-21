package network

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moemoeq/tyk-sre-app/internal/k8s"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	testing2 "k8s.io/client-go/testing"
)

func TestListNetworkPolicies(t *testing.T) {
	// Setup
	clientset := fake.NewSimpleClientset(&networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
	})
	k8sClient := &k8s.Client{Clientset: clientset}
	h := &Handler{K8sClient: k8sClient}

	// Create request
	req, err := http.NewRequest("GET", "/api/v1/network/policies", nil)
	assert.NoError(t, err)

	// Record response
	rr := httptest.NewRecorder()
	h.ListPolicies(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "test-policy")
}

func TestListNetworkPolicies_NamespaceFilter(t *testing.T) {
	// Setup with multiple namespaces
	clientset := fake.NewSimpleClientset(
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "policy-a",
				Namespace: "ns-a",
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "policy-b",
				Namespace: "ns-b",
			},
		},
	)
	k8sClient := &k8s.Client{Clientset: clientset}
	h := &Handler{K8sClient: k8sClient}

	// Create request filtering for ns-a
	req, err := http.NewRequest("GET", "/api/v1/network/policies?namespace=ns-a", nil)
	assert.NoError(t, err)

	// Record response
	rr := httptest.NewRecorder()
	h.ListPolicies(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "policy-a")
	assert.NotContains(t, rr.Body.String(), "policy-b")
}

func TestListNetworkPolicies_LabelSelector(t *testing.T) {
	// Setup with policies having different labels
	clientset := fake.NewSimpleClientset(
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "policy-a",
				Namespace: "default",
				Labels:    map[string]string{"app": "foo"},
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "policy-b",
				Namespace: "default",
				Labels:    map[string]string{"app": "bar"},
			},
		},
	)
	k8sClient := &k8s.Client{Clientset: clientset}
	h := &Handler{K8sClient: k8sClient}

	// Create request filtering for app=foo
	req, err := http.NewRequest("GET", "/api/v1/network/policies?labelSelector=app=foo", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.ListPolicies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// fake client in client-go might not fully support label filtering logic depending on version,
	// checking if it passes params correctly is the main goal here.
	// However, client-go fake usually supports basic label selection.
	// If it fails, we know handler logic is correct (passing param), implementation detail of fake client might vary.
	// Let's assume it works or just verifies the call.
	// Actually, client-go fake supports label selectors.
	assert.Contains(t, rr.Body.String(), "policy-a")
	assert.NotContains(t, rr.Body.String(), "policy-b")
}

func TestBlockWorkloads(t *testing.T) {
	// Setup
	clientset := fake.NewSimpleClientset()
	k8sClient := &k8s.Client{Clientset: clientset}
	h := &Handler{K8sClient: k8sClient}

	body := `{"target_a": {"namespace": "ns-a", "label_selector": "app=foo"}, "target_b": {"namespace": "ns-b", "label_selector": "app=bar"}}`
	req, err := http.NewRequest("POST", "/api/v1/network/block", strings.NewReader(body))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.BlockWorkloads(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify policies created
	// We expect 2 policies created
	actions := clientset.Actions()
	createActions := 0
	for _, action := range actions {
		if action.GetVerb() == "create" && action.GetResource().Resource == "networkpolicies" {
			createActions++
			pol := action.(testing2.CreateAction).GetObject().(*networkingv1.NetworkPolicy)
			// Basic check
			if pol.Namespace == "ns-a" {
				assert.Contains(t, pol.Name, "block-from-ns-b")
			} else if pol.Namespace == "ns-b" {
				assert.Contains(t, pol.Name, "block-from-ns-a")
			}
		}
	}
	assert.Equal(t, 2, createActions)
}

func TestUnblockWorkloads(t *testing.T) {
	// Calculate expected names
	hashA := hashLabel("app=foo")
	hashB := hashLabel("app=bar")

	policyNameA := "block-from-ns-b-" + hashB
	policyNameB := "block-from-ns-a-" + hashA

	// Setup with existing policies having correct names
	clientset := fake.NewSimpleClientset(
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: policyNameA, Namespace: "ns-a"}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: policyNameB, Namespace: "ns-b"}},
	)
	k8sClient := &k8s.Client{Clientset: clientset}
	h := &Handler{K8sClient: k8sClient}

	body := `{"target_a": {"namespace": "ns-a", "label_selector": "app=foo"}, "target_b": {"namespace": "ns-b", "label_selector": "app=bar"}}`
	req, err := http.NewRequest("DELETE", "/api/v1/network/block", strings.NewReader(body))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.UnblockWorkloads(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify actions
	actions := clientset.Actions()
	deleteActions := 0
	for _, action := range actions {
		if action.GetVerb() == "delete" && action.GetResource().Resource == "networkpolicies" {
			deleteActions++
		}
	}
	assert.Equal(t, 2, deleteActions)
}

func TestDeletePolicy(t *testing.T) {
	// Setup
	clientset := fake.NewSimpleClientset(&networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
	})
	k8sClient := &k8s.Client{Clientset: clientset}
	h := &Handler{K8sClient: k8sClient}

	// Test case: Success
	req, err := http.NewRequest("DELETE", "/api/v1/network/policies?namespace=default&name=test-policy", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.DeletePolicy(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify deletion
	actions := clientset.Actions()
	deleteActions := 0
	for _, action := range actions {
		if action.GetVerb() == "delete" && action.GetResource().Resource == "networkpolicies" {
			deleteActions++
			delAction := action.(testing2.DeleteAction)
			assert.Equal(t, "test-policy", delAction.GetName())
			assert.Equal(t, "default", delAction.GetNamespace())
		}
	}
	assert.Equal(t, 1, deleteActions)

	// Test case: Delete by UID
	// Need to setup client with UID
	uidPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "uid-policy",
			Namespace: "default",
			UID:       "12345",
		},
	}
	clientsetUID := fake.NewSimpleClientset(uidPolicy)
	k8sClientUID := &k8s.Client{Clientset: clientsetUID}
	hUID := &Handler{K8sClient: k8sClientUID}

	reqUID, err := http.NewRequest("DELETE", "/api/v1/network/policies?uid=12345", nil)
	assert.NoError(t, err)

	rrUID := httptest.NewRecorder()
	hUID.DeletePolicy(rrUID, reqUID)

	assert.Equal(t, http.StatusNoContent, rrUID.Code)

	// Verify deletion
	actionsUID := clientsetUID.Actions()
	deleteActionsUID := 0
	for _, action := range actionsUID {
		if action.GetVerb() == "delete" && action.GetResource().Resource == "networkpolicies" {
			deleteActionsUID++
			delAction := action.(testing2.DeleteAction)
			assert.Equal(t, "uid-policy", delAction.GetName())
		}
	}
	assert.Equal(t, 1, deleteActionsUID)

	// Test case: Not Found (UID)
	reqNotFoundUID, _ := http.NewRequest("DELETE", "/api/v1/network/policies?uid=99999", nil)
	rrNotFoundUID := httptest.NewRecorder()
	hUID.DeletePolicy(rrNotFoundUID, reqNotFoundUID)
	assert.Equal(t, http.StatusNotFound, rrNotFoundUID.Code)

	// Test case: Not Found (Name)
	reqNotFoundName, _ := http.NewRequest("DELETE", "/api/v1/network/policies?namespace=default&name=non-existent", nil)
	rrNotFoundName := httptest.NewRecorder()
	h.DeletePolicy(rrNotFoundName, reqNotFoundName)
	assert.Equal(t, http.StatusNotFound, rrNotFoundName.Code)

	// Test case: Missing parameters
	reqMissing, _ := http.NewRequest("DELETE", "/api/v1/network/policies?namespace=default", nil)
	rrMissing := httptest.NewRecorder()
	h.DeletePolicy(rrMissing, reqMissing)
	assert.Equal(t, http.StatusBadRequest, rrMissing.Code)
}

func TestParseLabelSelector(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "Single label",
			input:    "app=foo",
			expected: map[string]string{"app": "foo"},
		},
		{
			name:     "Multiple labels",
			input:    "app=foo,env=prod",
			expected: map[string]string{"app": "foo", "env": "prod"},
		},
		{
			name:     "Multiple labels with spaces",
			input:    " app = foo , env = prod ",
			expected: map[string]string{"app": "foo", "env": "prod"},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLabelSelector(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
