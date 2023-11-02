Your service ${target.k8s.deployment} is allowed to use more cpu than defined in ```request``` as this is only used for pod scheduling. Therefore, you should configure an upper limit to prevent using the entire cpu of the node.

[Kubernetes Documentation - Managing Container Resources](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
