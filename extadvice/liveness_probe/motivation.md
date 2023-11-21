Applications running for long periods may eventually end up in a broken state (e.g., a deadlock).
In this case, the application can't recover independently, whereas a simple restart already helps alleviate symptoms.
For detecting this, Kubernetes can probe liveness to check whether your container ${target.steadybit.label} is still working.


### Read More
[Kubernetes Documentation - Configure Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
