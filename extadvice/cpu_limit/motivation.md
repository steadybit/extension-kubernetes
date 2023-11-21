Your Kubernetes pods of ${target.steadybit.label} can use more CPU than defined in ```request``` as Kubernetes uses this only for scheduling pods.
Hence, you should configure an upper limit to prevent using the entire CPU of the node at cost of other pods.

### Read More
[Kubernetes Documentation - Managing Container Resources](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
