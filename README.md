<img src="./logo.png" height="130" align="right" alt="Kubernetes logo depicting a helm next to text 'Kubernetes'">

# Steadybit extension-kubernetes

A [Steadybit](https://www.steadybit.com/) extension implementation for Kubernetes.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.github.steadybit.extension_kubernetes).

## Configuration

| Environment Variable                          | Helm value               | Meaning                            | required |
|-----------------------------------------------|--------------------------|------------------------------------|----------|
| `STEADYBIT_EXTENSION_KUBERNETES_CLUSTER_NAME` | `kubernetes.clusterName` | The name of the kubernetes cluster | yes      |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

The process requires access rights to interact with the Kubernetes API.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: extension-kubernetes
rules:
  - apiGroups:
      - apps
    resources:
      - deployments
      - replicasets
      - daemonsets
      - statefulsets
    verbs:
      - get
      - list
      - watch
      - patch
  - apiGroups: [""]
    resources:
      - services
      - pods
      - nodes
      - events
    verbs:
      - get
      - list
      - watch
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: extension-kubernetes
  namespace: steadybit-extension
automountServiceAccountToken: true
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: extension-kubernetes
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: extension-kubernetes
subjects:
  - kind: ServiceAccount
    name: extension-kubernetes
    namespace: steadybit-extension
```

## Installation

We recommend that you deploy the extension with
our [official Helm chart](https://github.com/steadybit/extension-kubernetes/tree/main/charts/steadybit-extension-kubernetes).

### Helm

```sh
helm repo add steadybit https://steadybit.github.io/extension-kubernetes
helm repo update

helm upgrade steadybit-extension-kubernetes \\
  --install \\
  --wait \\
  --timeout 5m0s \\
  --create-namespace \\
  --namespace steadybit-extension \\
  steadybit/steadybit-extension-kubernetes
```

### Docker

You may alternatively start the Docker container manually.

```sh
docker run \\
  --env STEADYBIT_LOG_LEVEL=info \\
  --expose 8088 \\
  ghcr.io/steadybit/extension-kubernetes:latest
```

## Register the extension

Make sure to register the extension at the steadybit platform. Please refer to
the [documentation](https://docs.steadybit.com/integrate-with-steadybit/extensions/extension-installation) for more information.