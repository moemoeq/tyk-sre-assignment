# tyk-sre-app

# Go Project

Original Location: https://github.com/TykTechnologies/tyk-sre-assignment/tree/main/golang


## Development (Recommended)

The recommended way to run the application for development is using Docker Compose. This ensures a consistent environment with hot-reloading enabled (via [Air](https://github.com/cosmtrek/air)).

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

### Creating Local Kubernetes Cluster

```bash
kind create cluster --name local-dev-cluster
```
