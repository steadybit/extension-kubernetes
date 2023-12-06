Configure the ```podAntiAffinity``` properly to achieve spreading across multiple nodes, e.g.:

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
  % startHighlight %
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
              matchExpressions:
                - key: app
                  operator: In
                  values:
                    - example
                  topologyKey: "kubernetes.io/hostname"
  % endHighlight %
      containers:
        - name: ${target.steadybit.label}
          image: images.my-company.example/app:v4
```

### Read More
[Kubernetes Documentation - Assigning Pods to Nodes](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)

---
This advice is powered by [kube-score](https://kube-score.com/).
