Specify ${target.steadybit.label}&apos;s requested memory by defining the `requets` property in your Kubernetes manifest.

```yaml
apiVersion: v1
kind: Pod
metadata:
spec:
  containers:
    - name: example
      image: images.my-company.example/app:v4
      resources:
  % startHighlight %
        requests:
          memory: "64Mi"
  % endHighlight %
          cpu: "250m"
          ephemeral-storage: "2Gi"
        limits:
          memory: "128Mi"
          cpu: "500m"
          ephemeral-storage: "4Gi"
```

### Read More
[Kubernetes Documentation - Managing Container Resources](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)

---
This advice is powered by [kube-score](https://kube-score.com/).
