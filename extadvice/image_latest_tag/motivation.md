Relying on the ```:latest```-tag makes it hard to spot the exact version running in a cluster and impossible to perform downgrades.
In Docker, the ```:latest```-tag is just a default version which is not changed if you build a docker image with an explicit version-tag specified.

### Read More
- [Kubernetes Documentation - Container Images](https://kubernetes.io/docs/concepts/configuration/overview/#container-images)
- [Blog Post - What&apos;s Wrong With The Docker :latest Tag?](https://vsupalov.com/docker-latest-tag/)
