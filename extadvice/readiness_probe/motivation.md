Readiness probes are designed to indicate that an application has reached a &quot;ready&quot; state and can handle incoming requests.
Right now, pods of ${target.steadybit.label} immediately receive traffic, even when still starting up.


### Read More
[Kubernetes Documentation - Configure Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
