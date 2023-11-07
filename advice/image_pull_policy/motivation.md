By default, a node attempting to run a container only pulls the image if it isn&apos;t already cached. Consequently, the node may execute an older cached version of the image instead of the latest one. This results in having different versions running as well as access to an image without checking for correct ```ImagePullSecret```.

[Kubernetes Documentation - Images](https://kubernetes.io/docs/concepts/containers/images/)
