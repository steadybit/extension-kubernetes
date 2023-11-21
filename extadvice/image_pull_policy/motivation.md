By default, a node attempting to run a container only pulls the image if it isn&apos;t already cached.
Consequently, the node may execute an older cached version of the image instead of the latest one which leads to running different versions in the cluster.

### Read more
[Kubernetes Documentation - Images](https://kubernetes.io/docs/concepts/containers/images/)
