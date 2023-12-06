Specify the `readinessProbe` in your Kubernetes manifest appropriately for your type of application.
It can be as simple as an HTTP endpoint returning an HTTP status between 200 and 400 to indicate that the container running the application has started
successfully.

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    test: readiness
  name: readiness-http
spec:
  containers:
    - name: readiness
      image: k8s.gcr.io/readiness
      args:
        - /server
  % startHighlight %
    readinessProbe:
    httpGet:
      path: /health
      port: 8080
    initialDelaySeconds: 3
    periodSeconds: 3
  % endHighlight %
```

### Read More

[Kubernetes Documentation - Configure Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
