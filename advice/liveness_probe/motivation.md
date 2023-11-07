Applications running for a long periods of time eventually end up in a broken state (e.g. a deadlock). From this state the application can not recover and a simple restart helps to alleviate symptoms. For detecting this, kubernetes can probe liveness to detect whether your container ${target.k8s.deployment} is still working.

[Kubernetes Documentation - Configure Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
