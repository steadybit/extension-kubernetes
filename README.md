<img src="./logo.png" height="130" align="right" alt="Kubernetes logo depicting a helm next to text 'Kubernetes'">

# Steadybit extension-kubernetes

A [Steadybit](https://www.steadybit.com/) extension implementation for Kubernetes.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_kubernetes).

## Configuration

| Environment Variable                                             | Helm value                                  | Meaning                                                                                                                                                            | required | default                                                              |
|------------------------------------------------------------------|---------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|----------------------------------------------------------------------|
| `STEADYBIT_EXTENSION_KUBERNETES_CLUSTER_NAME`                    | `kubernetes.clusterName`                    | The name of the kubernetes cluster                                                                                                                                 | yes      |                                                                      |
| `STEADYBIT_EXTENSION_DISABLE_DISCOVERY_EXCLUDES`                 | `discovery.disableExcludes`                 | Ignore discovery excludes specified by `steadybit.com/discovery-disabled`                                                                                          | false    | `false`                                                              |
| `STEADYBIT_EXTENSION_LABEL_FILTER`                               |                                             | These labels will be ignored and not added to the discovered targets                                                                                               | false    | `controller-revision-hash,pod-template-generation,pod-template-hash` |
| `STEADYBIT_EXTENSION_ACTIVE_ADVICE_LIST`                         |                                             | List of active advice definitions, default is all (*). You can define a list of active adviceDefinitionId. See UI -> Settings -> Extension -> Advice -> Column: ID | false    | `*`                                                                  |
| `STEADYBIT_EXTENSION_ADVICE_SINGLE_REPLICA_MIN_REPLICAS`         |                                             | Minimal required replicas for the "Redundant Pod" advice                                                                                                           | false    | 2                                                                    |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_CONTAINER`    | `discovery.attributes.excludes.container`   | List of Target Attributes which will be excluded during container discovery. Checked by key equality and supporting trailing "*"                                   | false    |                                                                      |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_DEPLOYMENT`   | `discovery.attributes.excludes.deployment`  | List of Target Attributes which will be excluded during deployment discovery. Checked by key equality and supporting trailing "*"                                  | false    |                                                                      |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_DAEMON_SET`   | `discovery.attributes.excludes.daemonSet`   | List of Target Attributes which will be excluded during daemonSet discovery. Checked by key equality and supporting trailing "*"                                   | false    |                                                                      |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_STATEFUL_SET` | `discovery.attributes.excludes.statefulSet` | List of Target Attributes which will be excluded during statefulSet discovery. Checked by key equality and supporting trailing "*"                                 | false    |                                                                      |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_POD`          | `discovery.attributes.excludes.pod`         | List of Target Attributes which will be excluded during pod discovery. Checked by key equality and supporting trailing "*"                                         | false    |                                                                      |
| `STEADYBIT_EXTENSION_DISCOVERY_MAX_POD_COUNT`                    | `discovery.maxPodCount`                     | Skip listing pods, containers and hosts for deployments, statefulsets, etc. if there are more then the given pods.                                                 | false    | 50                                                                   |
| `STEADYBIT_EXTENSION_NAMESPACE`                                  | `Release.Namespace`                         | The namespace of the extension. If env var is set, discovery is only discovering in that namespace                                                                 | false    | `default`                                                            |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Permissions
The process requires access rights to interact with the Kubernetes API.

Please have a look at [/charts/steadybit-extension-kubernetes/templates/clusterrole.yaml](/charts/steadybit-extension-kubernetes/templates/clusterrole.yaml) for a recent list of required permission

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: extension-kubernetes
rules:
  - apiGroups:
    - ...
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

### Kubernetes

Detailed information about agent and extension installation in kubernetes can also be found in
our [documentation](https://docs.steadybit.com/install-and-configure/install-agent/install-on-kubernetes).

#### Recommended (via agent helm chart)

All extensions provide a helm chart that is also integrated in the
[helm-chart](https://github.com/steadybit/helm-charts/tree/main/charts/steadybit-agent) of the agent.

The extension is installed by default when you install the agent.

You must provide additional values to configure this extension.

```
--set extension-kubernetes.kubernetes.clusterName=<NAME_OF_YOUR_CLUSTER> \
```

Additional configuration options can be found in
the [helm-chart](https://github.com/steadybit/extension-kubernetes/blob/main/charts/steadybit-extension-kubernetes/values.yaml) of the
extension.

#### Alternative (via own helm chart)

If you need more control, you can install the extension via its
dedicated [helm-chart](https://github.com/steadybit/extension-kubernetes/blob/main/charts/steadybit-extension-kubernetes).

```bash
helm repo add steadybit-extension-kubernetes https://steadybit.github.io/extension-kubernetes
helm repo update
helm upgrade steadybit-extension-kubernetes \
  --install \
  --wait \
  --timeout 5m0s \
  --create-namespace \
  --namespace steadybit-agent \
  --set kubernetes.clusterName=<NAME_OF_YOUR_CLUSTER> \
  steadybit-extension-kubernetes/steadybit-extension-kubernetes
```

## Extension registration

Make sure that the extension is registered with the agent. In most cases this is done automatically. Please refer to
the [documentation](https://docs.steadybit.com/install-and-configure/install-agent/extension-registration) for more
information about extension registration and how to verify.

## mark resources as "do not discover"

to exclude a deployment / namespace / pod from discovery you can add the label `"steadybit.com/discovery-disabled": "true"` to the resource labels
