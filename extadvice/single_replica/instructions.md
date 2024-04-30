Kubernetes supports the concept of replicas to support redundancy for your pods.
You need to specify the desired replicas directly in the Kubernetes deployment or, in case your deployment is targeted by a HorizontalPodAutoScaler, in this workload resource.

*If you increase the replica, we strongly advise you to check if your application supports this.*

## Deployments Without HorizontalPodAutoScaler
When using a deployment without a HorizontalPodAutoScaler, you can specify the replicas directly within the deployment's manifest.
Under the hood, the deployment manages a ReplicaSet to control the desired redundancy.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  labels:
    name: ${target.attr('steadybit.label')}
spec:
% startHighlight %
  # modify min-replicas to match your use case, but should be >1
  replicas: 2
% endHighlight %
  selector:
    matchLabels:
      tier: ${target.attr('steadybit.label')}
```

[Kubernetes Documentation - Deployment Replicas](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#replicas)

## Deployments Targeted by HorizontalPodAutoScaler
When a HorizontalPodAutoscaler targets your deployment, you should adjust the `minReplicas` property of the HorizontalPodAutoscaler and omit the `replica` property in the deployment specification.
```yaml
apiVersion: apps/v1
kind: HorizontalPodAutoscaler
metadata:
  name: example
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ${target.attr('steadybit.label')}
% startHighlight %
  # modify min-replicas to match your use case, but should be >1
  minReplicas: 2
% endHighlight %
  maxReplicas: 10
```
[Kubernetes Documentation - Horizontal Pod Autoscaling](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
---
This advice is powered by [kube-score](https://kube-score.com/).
