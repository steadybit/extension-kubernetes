Makes sure that the ```podAntiAffinity``` is properly configured, e.g.:

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
							topologyKey: "kubernetes.io/hostname"
% endHighlight %
		containers:
			- name: ${target.k8s.deployment}
				image: images.my-company.example/app:v4
```
