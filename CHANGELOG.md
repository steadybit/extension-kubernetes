# Changelog

## v2.5.10

- Fixed advice's experiment for multi-availability zones to consistently use the same zone in every step (`single-azure-zone`, `single-aws-zone`, and `single-gcp-zone`
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
