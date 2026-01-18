package v1

import (
	"net/http"
)

func (api *API) getDeployments(w http.ResponseWriter, r *http.Request) {
	detailed := r.URL.Query().Get("detailed") == "true"
	deployments, err := api.K8sClient.ListDeployments(r.Context(), "")
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

		// Health Logic:
		// 1. ReadyReplicas must match Desired Replicas
		// 2. UpdatedReplicas must match Desired Replicas (Rolling update finished)
		// 3. UnavailableReplicas must be 0
		// 4. Status.conditions last status must be True
		if d.Status.ReadyReplicas == desired &&
			d.Status.UpdatedReplicas == desired &&
			d.Status.UnavailableReplicas == 0 &&
			len(d.Status.Conditions) > 0 &&
			d.Status.Conditions[len(d.Status.Conditions)-1].Status == "True" {
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
