When one pod is failing, your service ${target.k8s.deployment} will still be available, because there are more than 1 pod to handle the workload.
