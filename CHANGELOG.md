# Changelog

## v2.6.9 (Next)

- Discovery, Pod Count Check and Set Scale Action for ReplicaSets

## v2.6.8

- Improve the experiment suggestion of the memory limit advice

## v2.6.7

- Resync internal k8s cache every 10m and increase update debounce to 20s (both values are configurable)
- Optimize advice generation
- Updated dependencies

## v2.6.6

- possibility to disable the advice / kubescore feature
- Updated dependencies
- Updated go version to 1.24.4

## v2.6.5

- HAProxy Ingress: Discovery, delay and block traffic attack

## v2.6.4

- added "Set Image" attack that allows to set the image of a container in a deployment
- added the namespace to the messages of "Pod Count Check"
- Updated dependencies

## v2.6.3

- add more prefill-queries
- rollout restart attack check for already running rollout restart

## v2.6.2

- Updated dependencies

## v2.6.1

- Update dependencies
- Support wall-clock changes

## v2.6.0
- Removed the advice `single_aws_zone`,`single_azure_zone` and `single_gcp_zone` and combined them using the generic attribute `k8s.label.topology.kubernetes.io/zone`. With the new advice, you are no longer required to install the cloud provider specific extension.
  - If you like to migrate your existing advice state, like created experiments and you are running ON-Premise, you can use the following migration script after installing the new version of the extension:
    ```sql
    update sb_onprem.advice
    set advice_definition_id='com.steadybit.extension_kubernetes.advice.single-zone',
        validation_states   = replace(validation_states::text, 'com.steadybit.extension_kubernetes.single-aws-zone',
                                      'com.steadybit.extension_kubernetes.single-zone')::jsonb
    where advice_definition_id = 'com.steadybit.extension_kubernetes.advice.single-aws-zone';

    update sb_onprem.advice
    set advice_definition_id='com.steadybit.extension_kubernetes.advice.single-zone',
        validation_states   = replace(validation_states::text, 'com.steadybit.extension_kubernetes.single-azure-zone',
                                      'com.steadybit.extension_kubernetes.single-zone')::jsonb
    where advice_definition_id = 'com.steadybit.extension_kubernetes.advice.single-azure-zone';

    update sb_onprem.advice
    set advice_definition_id='com.steadybit.extension_kubernetes.advice.single-zone',
        validation_states   = replace(validation_states::text, 'com.steadybit.extension_kubernetes.single-gcp-zone',
                                      'com.steadybit.extension_kubernetes.single-zone')::jsonb
    where advice_definition_id = 'com.steadybit.extension_kubernetes.advice.single-gcp-zone';
    ```

## v2.5.21

- Update dependencies
- Changed labels for selection templates

## v2.5.20

- Integrated support for experiment templates in Advice to ease service's validation
- Fixed a bug for Azure and GCP, where DaemonSets aren't considered in an Advice

## v2.5.19

- Avoid unnecessary enrichment rules for node labels, improving performance
- update dependencies
- Use uid instead of name for user statement in Dockerfile

## v2.5.18

- Update dependencies (go 1.23)

## v2.5.17

- Update dependencies

## v2.5.16

- Increased timeout in the experiment for the single zone advice to detect a pod as being down within 45 seconds instead of just 30 seconds

## v2.5.15

- Be able to install the extension with a role instead of a service account to be able to work only in one namespace
  Example installation:
	```bash
	helm upgrade steadybit-agent --install --namespace <replace-me-with-namespace> \
  --create-namespace \
  --set agent.key="<replace-me>" \
  --set global.clusterName="<replace-me>" \
  --set extension-container.container.runtime="<replace-me>" \
  --set agent.registerUrl="<replace-me>"\
  --set rbac.roleKind="role" \
  --set agent.extensions.autodiscovery.namespace="<replace-me-with-namespace>" \
  --set extension-kubernetes.role.create=true \
  --set extension-kubernetes.roleBinding.create=true \
  --set extension-kubernetes.clusterRole.create=false \
  --set extension-kubernetes.clusterRoleBinding.create=false \
  steadybit/steadybit-agent
  ```

## v2.5.14

- Update dependencies

## v2.5.13

- Populate all k8s.node labels to host target type.
- Update dependencies


## v2.5.12

- Renamed "Pod Count Check" to "(Deployment, StatefulSet, DaemonSet) Pod Count Check"
- Pod-Targets now have a unique id. (Used by the UI to fetch details for a specific pod)
- Update dependencies

## v2.5.11

- Update dependencies (go 1.22)
- Added "Pod Count Check" for StatefulSets and DaemonSets
- Improved advice's experiment for multi availability zones (`single-azure-zone`, `single-aws-zone`, and `single-gcp-zone`) to establish a 20s base-line in the beginning of the experiment
- Add namespace label to container, k8s-container, k8s-deployment, k8s-statefulset and k8s-daemonset
- Use FreeMarker syntax for advice templates.
- Ignore Pods not in state "Running" in all discoveries

## v2.5.10

- Fixed advice's experiment for multi availability zones (`single-azure-zone`, `single-aws-zone`, and `single-gcp-zone`) to consistently use the same zone in every step
- Improved instruction text for advice `k8s-single-replica` to better explain how to increase replicas for deployments and HorizontalPodAutoscaler

## v2.5.9

 - Update dependencies
 - Remove some attributes which have been used by the old 'weakspot' feature
 - Clarify the log message, if the extension stops listing pods, containers and hosts for deployments, statefulsets, etc. because of the `discovery.maxPodCount` configuration

## v2.5.8

 - Update dependencies
 - feat: add `host.domainname` attribute containing the host FQDN

## v2.5.7

- Update dependencies
- fix: update deployments if services/hpas have changes
- fix: integrate kubescore check `horizontalpodautoscaler-replicas`

## v2.5.6

- Update dependencies

## v2.5.5

- use TargetEnrichmentRule Matcher Regex for copying k8s.label.* to container (exclude k8s.label.topology.*) (needs platform version >= 2.0.0 and agent version >= 2.0.2)

## v2.5.4

- Crash Loop Attack: validate specified container name with spec
- Crash Loop Attack: ignore when to be killed container is already gone
- Renamed attribute `k8s.deployment.replicas` to `k8s.specification.replicas`
- Update dependencies
- Add attributes `k8s.label.topology.kubernetes.io/zone`, `k8s.label.topology.kubernetes.io/region`, `k8s.label.node.kubernetes.io/instance-type`, `k8s.label.kubernetes.io/os` and `k8s.label.kubernetes.io/arch` to container, host, k8s-container, k8s-deplyoment, k8s-statefulset and k8s-daemonset

## v2.5.3

- Update extension-kit dependency to prevent a concurrent map write error

## v2.5.2

- invalid

## v2.5.1

- Update dependencies

## v2.5.0

- Discoveries added
  - pods
  - daemonsets
  - statefulsets
  - nodes
- Attack 'Delete Pod' added - :exclamation: Requires new permission `delete` for `pods` resources
- Attack 'Drain node' added - :exclamation: Requires new permission `create` for `pods/eviction` resources and `patch` for `nodes` resources
- Attack 'Taint node' added - :exclamation: Requires new permission `patch` for `nodes` resources
- Attack 'Scale Deployment' added - :exclamation: Requires new permission  `get`, `update` and `patch` for `deployments/scale` resources
- Attack 'Scale StatefulSet' added - :exclamation: Requires new permission `get`, `update` and `patch` for `statefulsets/scale` resources
- Attack 'Cause Crash Loop' added - :exclamation: Requires new permission `create` for `pod/exec` resources
- Added options to check if a pod count increased or decreased to the existing pod count check action
- Performance - Add hostnames to `kubernetes-deployment` during discovery instead of adding it via enrichment rule
- Performance - Enrich hosts via `kubernetes-node` instead of frequent enrichments via `kubernetes-container`
- Added `pprof` endpoints for debugging purposes
- Memory optimizations
- Removed the attribute `k8s.container.ready` as this causes unnecessary enrichment noise
- Added additional attributes to support advice / weakspots - :exclamation: Requires new permission `get`, `list`, and `watch` for `horizontalpodautoscalers` resources

## v2.4.2

- Possibility to exclude attributes from discovery

## v2.4.1

- fix k8s.service.name attribute incorrect for containers in multiple services

## v2.4.0

- `kubernetes-container` are handled as enrichment data and not as targets anymore. (This requires at least agent 1.0.92 and platform 1.0.79)

## v2.3.6

- fix node count check config parsing

## v2.3.5

- update dependencies

## v2.3.4

- ignore container with label `steadybit.com.discovery-disabled"="true"` during discovery

## v2.3.3

- migration to new unified steadybit actionIds and targetTypes
- ignore all labeled deployments and containers from discovery

## v2.3.2

- fix node count check config parsing

## v2.3.1

- update dependencies

## v2.3.0

- Read only file system

## v2.2.0

- Code refactorings

## v2.1.1

- Kubernetes Event Log will now listen to a stop method and send the last messages before exiting

## v2.1.0

 - Kubernetes Event Log and Pod Metrics will need a cluster-selection to support multiple kubernetes clusters

## v2.0.0

 - Added Discoveries for Deployments and Container
 - Added Pod Count Check and Node Count check
 - Added Pod Count Metrics and Event Logs

## v1.3.0

 - Support creation of a TLS server through the environment variables `STEADYBIT_EXTENSION_TLS_SERVER_CERT` and `STEADYBIT_EXTENSION_TLS_SERVER_KEY`. Both environment variables must refer to files containing the certificate and key in PEM format.
 - Support mutual TLS through the environment variable `STEADYBIT_EXTENSION_TLS_CLIENT_CAS`. The environment must refer to a comma-separated list of files containing allowed clients' CA certificates in PEM format.

## v1.2.0

 - Support for the `STEADYBIT_LOG_FORMAT` env variable. When set to `json`, extensions will log JSON lines to stderr.

## v1.1.1

 - Rollout readiness check always fails when a timeout is specified.

## v1.1.0

 - upgrade `extension-kit` to support additional debugging log output.

## v1.0.0

 - Initial release
