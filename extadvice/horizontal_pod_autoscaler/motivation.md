The number of ${target.steadybit.label}&apos;s pods is configured to be scaled automatically by the horizontal pod autoscaler.
Even so, it&apos;s `DeploymentSpec` also contains a fixed `ReplicaSet`.


When applying ${target.steadybit.label}&apos;s specification, the number of pods will be reverted to the configured `replicas` of pods independent of the desired pod count of the horizontal pod autoscaler.
