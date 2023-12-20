When Kubernetes redeploys ${target.steadybit.label}, it can determine when a container is ready to accept incoming requests and thus avoids routing requests to the container too early.

If you also added a liveness probe, Kubernetes can detect erroneous pods and restart them in case of an error. Your service ${target.steadybit.label} will become available again.
