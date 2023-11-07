Readiness probes are designed to ensure that an application has reached a &quot;ready&quot; state. In many cases there is a period of time between when a webserver process starts and when it is ready to receive traffic. A readiness probe can ensure the traffic is not sent to your pod ${target.k8s.deployment} until it is actually ready to receive traffic.

[Kubernetes Documentation - Configure Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
