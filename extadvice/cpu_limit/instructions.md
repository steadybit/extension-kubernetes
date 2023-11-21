Specify ${target.steadybit.label}&apos;s CPU limit by defining the ```limits``` property in your kubernetes manifest.

```yaml
apiVersion: v1
kind: Pod
metadata:
spec:
  containers:
  - name: ${target.steadybit.label}
    image: images.my-company.example/app:v4
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
% startHighlight %
      limits:
        memory: "128Mi"
        cpu: "500m"
% endHighlight %
```

### Read More
[Kubernetes Documentation - Managing Container Resources](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
