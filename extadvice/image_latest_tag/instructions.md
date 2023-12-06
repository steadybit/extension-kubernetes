Makes sure that your Kubernetes manifest is based on an `image` with a fixed and explicit tag

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name:  ${target.steadybit.label}
% startHighlight %
      image: images.my-company.example/app:v4
% endHighlight %

```

### Read More
- [Kubernetes Documentation - Container Images](https://kubernetes.io/docs/concepts/configuration/overview/#container-images)
- [Blog Post - What&apos;s Wrong With The Docker :latest Tag?](https://vsupalov.com/docker-latest-tag/)
