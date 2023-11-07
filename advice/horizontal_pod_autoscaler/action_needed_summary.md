The service ${target.k8s.deployment} is scaled by a horizontal pod autoscaler, but a static
replica count is configured in the ```DeploymentSpec``` as well.

#### Kube-Score
Grade: ${target.k8s.kube-score.deployment-targeted-by-hpa-does-not-have-replicas-configured.grade}

${target.k8s.kube-score.deployment-targeted-by-hpa-does-not-have-replicas-configured.comment:normal}
