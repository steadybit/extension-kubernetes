When Kubernetes redeploys ${target.steadybit.label}, it can't determine when the following container are ready to accept incoming requests.
They may receive requests before being able to handle them properly.

**Container affected:** ${target.k8s.container.probes.readiness.not-set[]}
