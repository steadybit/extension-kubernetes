When your service ${target.k8s.deployment} uses too much cpu, it will be limited by the configured
CPU limit.

Specify the upper limit to be used by defining the ```limits``` property in your
kubernetes manifest:
```yaml
apiVersion: v1
kind: Pod
metadata:
spec:
  containers:
  - name: gateway
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
