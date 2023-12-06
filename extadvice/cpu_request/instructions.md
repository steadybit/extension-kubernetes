Specify ${target.steadybit.label}&apos;s requested CPU by defining the `requets` property in your Kubernetes manifest.

```yaml
apiVersion: v1
kind: Pod
metadata:
spec:
  containers:
    - name: ${target.steadybit.label}
      image: images.my-company.example/app:v4
      resources:
  % startHighlight %
        requests:
          memory: "64Mi"
          cpu: "250m"
  % endHighlight %
        limits:
          memory: "128Mi"
          cpu: "500m"
```

### Read More
[Kubernetes Documentation - Managing Container Resources](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
