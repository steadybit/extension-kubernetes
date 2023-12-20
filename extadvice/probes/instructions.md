Specify the `readinessProbe` in your Kubernetes manifest appropriately for your type of application.
It can be as simple as an HTTP endpoint returning an HTTP status between 200 and 400 to indicate that the container
running the application has started
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

You may optionally specify the `livenessProbe` in your Kubernetes manifest appropriately for your type of application.
It can be as simple as an HTTP endpoint returning an HTTP status between 200 and 400 to indicate that everything is
fine.

```yaml
apiVersion: v1
kind: Pod
metadata:
spec:
  containers:
    - name: example
      image: k8s.gcr.io/liveness
      args:
        - /server
% startHighlight %
      livenessProbe:
        httpGet:
          path: /health
          port: 8080
        initialDelaySeconds: 3
        periodSeconds: 3
% endHighlight %
```

Make sure to not use the same probes for readiness and liveness. If the liveness probe fails, the container will be
restarted, but if the readiness probe fails, the container will be removed from the service load balancer.

### Read More

[Kubernetes Documentation - Configure Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)

---
This advice is powered by [kube-score](https://kube-score.com/).
