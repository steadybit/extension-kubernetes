When your containers of the ${target.attr('k8s.workload-type')} *${target.attr('steadybit.label')}* don't specify requested memory, scheduling and autoscaling of Kubernetes works suboptimal.
<br/>
<br/>
**Affected Containers:** *<#list target.attrs('k8s.container.spec.request.memory.not-set') as item>${item}<#sep>, </#list>*
