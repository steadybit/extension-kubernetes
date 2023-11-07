Change ```ReplicaSet``` to two (or more) in your Kubernetes configuration in order to increase the scheduling of additional pods. The availability of your service ${target.k8s.deployment} will most likely improve.
```warning
If you increase the replica we strongly advice you to check if this is supported by your application.
```

```yaml
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: frontend
  labels:
    name: ${target.k8s.deployment}
spec:
% startHighlight %
  # modify replicas according to your case
  replicas: 2
% endHighlight %
  selector:
    matchLabels:
      tier: ${target.k8s.deployment}
```
