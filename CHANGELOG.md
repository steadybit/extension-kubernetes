# Changelog

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