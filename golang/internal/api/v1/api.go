package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/moemoeq/tyk-sre-app/internal/api/v1/network"
	"github.com/moemoeq/tyk-sre-app/internal/config"
	"github.com/moemoeq/tyk-sre-app/internal/k8s"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type API struct {
	Config    *config.Config
	K8sClient *k8s.Client
}

// EnrichedDeployment wraps appsv1.Deployment with health information.
type EnrichedDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              *appsv1.DeploymentSpec  `json:"spec,omitempty"`
	Status            appsv1.DeploymentStatus `json:"status"`
	Health            bool                    `json:"health"`
}

func New(cfg *config.Config, k8sClient *k8s.Client) *API {
	return &API{
		Config:    cfg,
		K8sClient: k8sClient,
	}
}

// Register API routes
func (api *API) Register(mux *http.ServeMux) {
	mux.Handle("/deployments", api.wrap(api.getDeployments))
	mux.Handle("/reachability", api.wrap(api.checkK8sReachability))

	// TODO: refactor network route into subrouter
	// Network Handlers
	netHandler := &network.Handler{K8sClient: api.K8sClient}
	mux.Handle("/network/policies", api.wrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			netHandler.ListPolicies(w, r)
		} else if r.Method == http.MethodDelete {
			netHandler.DeletePolicy(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.Handle("/network/block", api.wrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			netHandler.BlockWorkloads(w, r)
		} else if r.Method == http.MethodDelete {
			netHandler.UnblockWorkloads(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
}

func (api *API) wrap(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Common headers
		w.Header().Set("Content-Type", "application/json")

		h(w, r)
	})
}

func (a *API) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.WriteHeader(status)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			// logging
			fmt.Println("failed to encode response", err)
		}
	}
}

func (a *API) respondError(w http.ResponseWriter, status int, message string) {
	a.respondJSON(w, status, map[string]string{"error": message})
	if a.Config.Environment == "dev" {
		fmt.Println("response error", message)
	}
}
