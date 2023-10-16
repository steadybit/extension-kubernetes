# Instructions

When your service ${target.k8s.container.name} uses too much cpu, it will be limited by the configured
CPU limit.

Specify the upper limit to be used by defining the ```limits``` property in your
kubernetes manifest:
```${target.k8s.cpu.limit}```