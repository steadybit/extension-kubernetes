Make sure that the ```deploymentStrategyType``` is set to
``RollingUpdate``.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name:  ${target.k8s.deployment}
spec:
  replicas: 5
% startHighlight %
  strategy:
    type: RollingUpdate
    rollingUpdate:
    maxSurge: 3
    maxUnavailable: 0
% endHighlight %

```
