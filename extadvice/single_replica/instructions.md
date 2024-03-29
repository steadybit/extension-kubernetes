Change `ReplicaSet` to two (or more) in your Kubernetes configuration in order to increase the scheduling of additional pods. The availability of your service ${target.steadybit.label} will most likely improve.

```yaml
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: frontend
  labels:
    name: ${target.steadybit.label:normal}
spec:
% startHighlight %
  # modify replicas according to your case
  replicas: 2
% endHighlight %
  selector:
    matchLabels:
      tier: ${target.steadybit.label:normal}
```

If your Deployment is targeted by a HorizontalPodAutoscaler, you should adjust the `minReplicas` property of the HorizontalPodAutoscaler and omit the `replica` property in the Deployment specification.

**If you increase the replica we strongly advice you to check if this is supported by your application.**

---
This advice is powered by [kube-score](https://kube-score.com/).
