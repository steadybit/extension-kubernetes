When your service ${target.k8s.container.name} uses too much cpu, it will be limited by the configured
CPU limit.

Specify the upper limit to be used by defining the ```limits``` property in your
kubernetes manifest:
```
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
      <Code p={0} color={'coral'}>{`      limits:
        memory: "128Mi"
        cpu: "500m"`}</Code>
```
