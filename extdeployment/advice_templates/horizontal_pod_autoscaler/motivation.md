The number of pods of ${target.k8s.deployment} is configured to be scaled automatically by the
horizontal pod autoscaler. Even so, it&apos;s ```DeploymentSpec``` also contains a fixed ```ReplicaSet```.


When applying ${target.k8s.deployment}&apos;s specification, the number of pods will be reverted
to the configured ```replicas``` of pods independent of the desired pod count of the
horizontal pod autoscaler.

[Kubernetes Documentation - Horizontal Pod Autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
