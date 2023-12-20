Specify the `imagePullPolicy` with the value `Always` in your Kubernetes manifest.

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: exampe
      image: images.my-company.example/app:v4
% startHighlight %
      imagePullPolicy: Always
% endHighlight %
```

### Read more

[Kubernetes Documentation - Images Pull Policy](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy)

---
This advice is powered by [kube-score](https://kube-score.com/).
