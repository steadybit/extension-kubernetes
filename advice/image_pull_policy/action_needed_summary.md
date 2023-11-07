When ${target.k8s.deployment} will be redeployed, Kubernetes will possibly not update the container image. This may result in running different versions of ${target.k8s.deployment}.

#### Kube-Score
Grade: ${target.k8s.kube-score.container-image-pull-policy.grade}

${target.k8s.kube-score.container-image-pull-policy.comment:normal}
