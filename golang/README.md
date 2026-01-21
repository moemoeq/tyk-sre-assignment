# tyk-sre-app

# Go Project

Original Location: https://github.com/TykTechnologies/tyk-sre-assignment/tree/main/golang


## Development (Recommended)

The recommended way to run the application for development is using Docker Compose. This ensures a consistent environment with hot-reloading enabled (via [Air](https://github.com/cosmtrek/air)).


### Prepare Local Kubernetes Cluster

```bash
# If you have an existing kind cluster named 'kind-cluster'
kind get kubeconfig --name kind-cluster > .kind-kubeconfig

# Or create a new one
kind create cluster --name kind-cluster
kind get kubeconfig --name kind-cluster > .kind-kubeconfig
```

modify .kind-kubeconfig server field to reach the API server example `https://kind-cluster-control-plane:6443` 

Run the following command to start the development server:

```bash
docker-compose up --build
```

The application will be available at `http://localhost:8080`.

### Manual Build

If you prefer to build and run the application locally without Docker:

```bash
go mod tidy && go build -o tyk-sre-app ./cmd/server
```

To run it against a real Kubernetes API server:
```bash
./server --kubeconfig '/path/to/your/kube/conf' --address ":8080"
```

### Testing

To execute unit tests:
```bash
go test -v ./...
```

### API Request Example

```bash
# Check k8s reachability
curl http://localhost:8080/api/v1/reachability

# Get All Deployments
curl http://localhost:8080/api/v1/deployments

# Get All Deployments with detailed information
curl http://localhost:8080/api/v1/deployments?detailed=true

# Get Deployments by namespace
curl http://localhost:8080/api/v1/deployments?namespace=kube-system

# Get Deployments by label selector
curl http://localhost:8080/api/v1/deployments?labelSelector=k8s-app=kube-dns

# Get Deployments by field selector
curl http://localhost:8080/api/v1/deployments?fieldSelector=metadata.name=local-path-provisioner

# List Network Policies
curl http://localhost:8080/api/v1/network/policies

# List Network Policies with detailed information
curl http://localhost:8080/api/v1/network/policies?detailed=true

# List Network Policies by namespace
curl http://localhost:8080/api/v1/network/policies?namespace=kube-system

# Block workload
curl -v -X POST http://localhost:8080/api/v1/network/block \
-H "Content-Type: application/json" \
-d '{
  "target_a": {"namespace": "poc-ns-a", "label_selector": "app=foo"},
  "target_b": {"namespace": "poc-ns-b", "label_selector": "app=bar"}
}'
# Unblock workload
curl -v -X DELETE http://localhost:8080/api/v1/network/block \
-H "Content-Type: application/json" \
-d '{
  "target_a": {"namespace": "poc-ns-a", "label_selector": "app=foo"},
  "target_b": {"namespace": "poc-ns-b", "label_selector": "app=bar"}
}'

# DELETE Network Policy by name and namespace
curl -v -X DELETE "http://localhost:8080/api/v1/network/policies?namespace=poc-ns-b&name=deny-from-a"

# DELETE Network Policy by UID
 curl -v -X DELETE "http://localhost:8080/api/v1/network/policies?uid=bd1c9956-e33e-4358-af18-b374f0bc02e9"

```
