Schedule Kubernetes nodes in different availability zones and configure a `podAntiAffinity` to spread your pods across different zones.

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
            topologyKey: "topology.kubernetes.io/zone"
% endHighlight %
    containers:
      - name: example
        image: images.my-company.example/app:v4
```
