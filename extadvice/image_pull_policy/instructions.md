Specify the <Code inline>imagePullPolicy</Code> with the value ```Always``` in your kubernetes manifest.

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name:  ${target.steadybit.label}
      image: images.my-company.example/app:v4
% startHighlight %
			imagePullPolicy: Always
% endHighlight %

```
