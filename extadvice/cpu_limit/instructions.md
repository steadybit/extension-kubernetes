Specify *${target.attr('steadybit.label')}*&apos;s CPU limit by defining the `limits` property in your Kubernetes manifest.

```yaml
apiVersion: v1
kind: Pod
metadata:
spec:
  containers:
    - name: example
      image: images.my-company.example/app:v4
      resources:
        requests:
          memory: "64Mi"
          cpu: "250m"
          ephemeral-storage: "2Gi"
  % startHighlight %
        limits:
          cpu: "500m"
  % endHighlight %
          memory: "128Mi"
          ephemeral-storage: "4Gi"

```

### Read More

[Kubernetes Documentation - Managing Container Resources](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)

---
This advice is powered by [kube-score](https://kube-score.com/).
