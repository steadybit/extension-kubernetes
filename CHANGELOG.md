# Changelog


## v2.5.x (NEXT RELEASE)

- Discoveries added
  - pods
  - daemonsets
  - statefulsets
  - nodes
- Attack "Delete Pod" added - :exclamation: Requires new permission `delete` for `pods` resources
- Attack "Drain node" added - :exclamation: Requires new permission `create` for `pods/eviction` resources and `patch` for `nodes` resources
- Attack "Taint node" added - :exclamation: Requires new permission `patch` for `nodes` resources
- Performance - Add hostnames to kubernetes-deployment during discovery instead of adding it via enrichment rule
- Added `pprof` endpoints for debugging purposes
- Memory optimizations
- Removed the attribute `k8s.container.ready` as this causes unnecessary enrichment noise

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
