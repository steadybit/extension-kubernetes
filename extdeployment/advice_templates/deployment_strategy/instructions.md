# Instructions

Make sure that the <Code inline={true}>deploymentStrategyType</Code> is set to{' '}
<Code inline={true}>RollingUpdate</Code>.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name:  ${target.k8s.deployment}
spec:
  replicas: 5
		<Code inline color={'coral'}>
			{`  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 3
      maxUnavailable: 0
`}
		</Code>
```
