<img src="./logo.png" height="130" align="right" alt="Kubernetes logo depicting a helm next to text 'Kubernetes'">

# Steadybit extension-kubernetes

A [Steadybit](https://www.steadybit.com/) attack and check implementation for Kubernetes.

## Capabilities

 - Deployments
     - Attacks
         - Rollout restart (`kubectl rollout restart`) 
     - Checks
         - Deployment rollout status (`kubectl rollout status`)  

## Configuration

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

## Deployment

We recommend that you deploy the extension with our [official Helm chart](https://github.com/steadybit/helm-charts/tree/main/charts/steadybit-extension-kubernetes).

## Agent Configuration

The Steadybit Kubernetes agent needs to be configured to interact with the Kubernetes extension by adding the following environment variables:

```shell
# Make sure to adapt the URLs and indices in the environment variables names as necessary for your setup

STEADYBIT_AGENT_ACTIONS_EXTENSIONS_0_URL=http://steadybit-extension-kubernetes.steadybit-extension.svc.cluster.local:8088
STEADYBIT_AGENT_DISCOVERIES_EXTENSIONS_0_URL=http://steadybit-extension-kubernetes.steadybit-extension.svc.cluster.local:8088
```

When leveraging our official Helm charts, you can set the configuration through additional environment variables on the agent:

```
--set agent.env[0].name=STEADYBIT_AGENT_ACTIONS_EXTENSIONS_0_URL \
--set agent.env[0].value="http://steadybit-extension-kubernetes.steadybit-extension.svc.cluster.local:8088" \
--set agent.env[1].name=STEADYBIT_AGENT_DISCOVERIES_EXTENSIONS_0_URL \
--set agent.env[1].value="http://steadybit-extension-kubernetes.steadybit-extension.svc.cluster.local:8088"
```
