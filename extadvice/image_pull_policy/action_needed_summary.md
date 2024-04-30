On redeployment, Kubernetes may not update container images of the ${target.attr('k8s.workload-type')} *${target.attr('steadybit.label')}*, if it is already cached.
<br/>
<br/>
**Affected Containers:** *<#list target.attrs('k8s.container.image.without-image-pull-policy-always') as item>${item}<#sep>, </#list>*
