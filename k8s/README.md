# K8S Helm Chart for tyk-sre-app

This directory contains the Helm chart for deploying the `tyk-sre-app`.

## Prerequisites

- [Helm](https://helm.sh/docs/intro/install/) (v3+)
- Kubernetes cluster (e.g., this chart tested with Kind)
- `kubectl` configured to communicate with your cluster

## Installation

To install the chart with the release name `tyk-sre-app`:

```bash
helm upgrade --install tyk-sre-app ./tyk-sre-app --namespace default --create-namespace
```

## Developer Guide

For developers modifying the chart, the following commands are useful to verify changes before pushing.

### 1. Linting the Chart

Check the chart for possible issues and best practices:

```bash
helm lint ./tyk-sre-app
```

### 2. Debugging Templates

Render the templates locally to inspect the generated manifests without deploying. This is useful to verify logic in templates:

```bash
helm template tyk-sre-app ./tyk-sre-app --debug
```
