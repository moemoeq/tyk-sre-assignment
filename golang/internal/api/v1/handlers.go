package v1

import (
	"net/http"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (api *API) getDeployments(w http.ResponseWriter, r *http.Request) {
	detailed := r.URL.Query().Get("detailed") == "true"
	namespace := r.URL.Query().Get("namespace")
	labelSelector := r.URL.Query().Get("labelSelector")
	fieldSelector := r.URL.Query().Get("fieldSelector")

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}

	deployments, err := api.K8sClient.ListDeployments(r.Context(), namespace, listOptions)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]any, 0, len(deployments))
	for _, d := range deployments {
		isHealthy := false
		desired := int32(0)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}

		checkCondition := func(t appsv1.DeploymentConditionType) bool {
			return slices.ContainsFunc(d.Status.Conditions, func(c appsv1.DeploymentCondition) bool {
				return c.Type == t && c.Status == "True"
			})
		}
		// Health Logic:
		// 1. ReadyReplicas must match Desired Replicas
		// 2. UpdatedReplicas must match Desired Replicas (Rolling update finished)
		// 3. UnavailableReplicas must be 0
		// 4. Status.conditions type=Progressing must be True
		// 5. Status.conditions type=Available must be True

		if d.Status.ReadyReplicas == desired &&
			d.Status.UpdatedReplicas == desired &&
			d.Status.UnavailableReplicas == 0 &&
			checkCondition(appsv1.DeploymentAvailable) &&
			checkCondition(appsv1.DeploymentProgressing) {
			isHealthy = true
		}

		enrichment := EnrichedDeployment{
			TypeMeta:   d.TypeMeta,
			ObjectMeta: d.ObjectMeta,
			Status:     d.Status,
			Health:     isHealthy,
		}

		if detailed {
			enrichment.Spec = &d.Spec
		} else {
			// Filter noisy metadata if not detailed
			enrichment.ManagedFields = nil
		}

		response = append(response, enrichment)
	}

	api.respondJSON(w, http.StatusOK, response)
}

func (api *API) checkK8sReachability(w http.ResponseWriter, r *http.Request) {
	status := api.K8sClient.CheckConnectivity(r.Context())

	httpStatus := http.StatusOK
	if !status.Status {
		httpStatus = http.StatusServiceUnavailable
	}

	api.respondJSON(w, httpStatus, status)
}
