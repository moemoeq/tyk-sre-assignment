package network

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/moemoeq/tyk-sre-app/internal/k8s"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Handler struct {
	K8sClient *k8s.Client
}

// Helpers
func parseLabelSelector(s string) map[string]string {
	labels := make(map[string]string)
	if s == "" {
		return labels
	}
	parts := strings.Split(s, ",")
	for _, part := range parts {
		kv := strings.Split(strings.TrimSpace(part), "=")
		if len(kv) == 2 {
			labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return labels
}

func hashLabel(s string) string {
	hash, err := hashstructure.Hash(s, hashstructure.FormatV2, nil)
	if err != nil {
		// Fallback or panic, but for simple strings unlikely to fail
		return "error"
	}
	return fmt.Sprintf("%x", hash)
}

func generatePolicyName(namespace, labelSelector string) string {
	return "block-from-" + namespace + "-" + hashLabel(labelSelector)
}

func convertToNotIn(labels map[string]string) []metav1.LabelSelectorRequirement {
	var reqs []metav1.LabelSelectorRequirement
	for k, v := range labels {
		reqs = append(reqs, metav1.LabelSelectorRequirement{
			Key:      k,
			Operator: metav1.LabelSelectorOpNotIn,
			Values:   []string{v},
		})
	}
	return reqs
}

func convertToDoesNotExist(labels map[string]string) []metav1.LabelSelectorRequirement {
	var reqs []metav1.LabelSelectorRequirement
	for k := range labels {
		reqs = append(reqs, metav1.LabelSelectorRequirement{
			Key:      k,
			Operator: metav1.LabelSelectorOpDoesNotExist,
		})
	}
	return reqs
}

// ListPolicies handles GET requests to list NetworkPolicies
func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	detailed := r.URL.Query().Get("detailed") == "true"
	namespace := r.URL.Query().Get("namespace")

	// labelSelector := r.URL.Query().Get("labelSelector")
	// NOTE: filedSelector is not implemented because it is out of scope
	// fieldSelector := r.URL.Query().Get("fieldSelector")

	listOptions := metav1.ListOptions{
		LabelSelector: r.URL.Query().Get("labelSelector"),
	}

	policies, err := h.K8sClient.ListNetworkPolicies(r.Context(), namespace, listOptions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !detailed {
		for i := range policies {
			policies[i].ManagedFields = nil
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

type WorkloadTarget struct {
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"label_selector"`
}

type BlockRequest struct {
	TargetA WorkloadTarget `json:"target_a"`
	TargetB WorkloadTarget `json:"target_b"`
}

// Creates NetworkPolicies to block traffic between two workloads.
// Policies are created on both workloads.
// If the operation fails, attempt to rollback by deleting the created policies. (keep pair)

func (h *Handler) BlockWorkloads(w http.ResponseWriter, r *http.Request) {
	var req BlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	policyA := h.generateBlockPolicy(req.TargetA, req.TargetB)
	if _, err := h.K8sClient.CreateNetworkPolicy(r.Context(), policyA); err != nil {
		http.Error(w, "failed to create policy A: "+err.Error(), http.StatusInternalServerError)
		return
	}

	policyB := h.generateBlockPolicy(req.TargetB, req.TargetA)
	if _, err := h.K8sClient.CreateNetworkPolicy(r.Context(), policyB); err != nil {
		// TODO: implement much more elegant rollback
		// Delete Policy A (rollback)
		errA := h.K8sClient.DeleteNetworkPolicy(r.Context(), req.TargetA.Namespace, policyA.Name)
		if errA != nil {
			http.Error(w, "failed to create policy B and failed to rollback policy A(DELETE): "+errA.Error(), http.StatusInternalServerError)
			return
		}

		http.Error(w, "failed to create policy B: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "blocked"})
}

// Deletes the blocking NetworkPolicies.
// Policies should be deleted on both workloads.
func (h *Handler) UnblockWorkloads(w http.ResponseWriter, r *http.Request) {
	var req BlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// get policy names
	policyNameA := generatePolicyName(req.TargetB.Namespace, req.TargetB.LabelSelector)
	policyNameB := generatePolicyName(req.TargetA.Namespace, req.TargetA.LabelSelector)

	// Delete Policies
	if err := h.K8sClient.DeleteNetworkPolicy(r.Context(), req.TargetA.Namespace, policyNameA); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, "failed to delete policy A: "+err.Error(), status)
		return
	}
	// TODO: Decide to implement or not rollback process(do or donot?)
	if err := h.K8sClient.DeleteNetworkPolicy(r.Context(), req.TargetB.Namespace, policyNameB); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, "failed to delete policy B: "+err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "unblocked"})
}

// Helper to generate "Allow All Except" NetworkPolicy
// WHY? K8S NetworkPolicy doesn't support "deny" policy only works "allow" based
// so we need to create "Allow All Except" policy
func (h *Handler) generateBlockPolicy(target, blocked WorkloadTarget) *networkingv1.NetworkPolicy {
	// Parse label selectors
	targetLabels := parseLabelSelector(target.LabelSelector)
	blockedLabels := parseLabelSelector(blocked.LabelSelector)

	policyName := generatePolicyName(blocked.Namespace, blocked.LabelSelector)

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: target.Namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: targetLabels,
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						// Rule 1: Allow any Namespace NOT equal to blocked.Namespace
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "kubernetes.io/metadata.name",
										Operator: metav1.LabelSelectorOpNotIn,
										Values:   []string{blocked.Namespace},
									},
								},
							},
						},
						// Rule 2: Allow same Namespace (blocked.Namespace) BUT NOT blocked labels
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"kubernetes.io/metadata.name": blocked.Namespace},
							},
							PodSelector: &metav1.LabelSelector{
								MatchExpressions: convertToNotIn(blockedLabels),
							},
						},
						// Rule 3: Allow same Namespace (blocked.Namespace) BUT DoesNotExist (key missing)
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"kubernetes.io/metadata.name": blocked.Namespace},
							},
							PodSelector: &metav1.LabelSelector{
								MatchExpressions: convertToDoesNotExist(blockedLabels),
							},
						},
					},
				},
			},
		},
	}
}

// for manual deletion policy by name or UID.
func (h *Handler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")

	// Delete by UID
	// just UID, or (optional) Namespace + UID
	if uid != "" {
		if err := h.K8sClient.DeleteNetworkPolicyByUID(r.Context(), namespace, uid); err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			http.Error(w, fmt.Sprintf("failed to delete policy by UID %s: %v", uid, err), status)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Delete by Name & Namespace (requires both can uniquely ident)
	if namespace == "" || name == "" {
		http.Error(w, "namespace and name are required query parameters (or provide uid)", http.StatusBadRequest)
		return
	}

	if err := h.K8sClient.DeleteNetworkPolicy(r.Context(), namespace, name); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, fmt.Sprintf("failed to delete policy %s/%s: %v", namespace, name, err), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
