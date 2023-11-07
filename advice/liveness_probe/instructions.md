Specify the ```livenessProbe``` in your kubernetes manifest appropriately for your type of application. A simple example could be a HTTP endpoint which returns a HTTP status between 200 and 400 to indicate that everything is fine.

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    test: liveness
  name: liveness-http
spec:
  containers:
    - name: liveness
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
