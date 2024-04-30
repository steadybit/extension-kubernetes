Readiness probes are designed to indicate that a container has reached a &quot;ready&quot; state and can handle incoming requests.
Right now, some containers of *${target.attr('steadybit.label')}* immediately receive traffic, even when still starting up.

Applications running for long periods may eventually end up in a broken state (e.g., a deadlock).
In this case, the application can't recover independently, whereas a simple restart already helps alleviate symptoms.
For detecting this, Kubernetes can probe liveness to check whether your container *${target.attr('steadybit.label')}* is still working.
