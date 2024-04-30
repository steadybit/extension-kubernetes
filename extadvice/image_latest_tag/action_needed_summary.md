Some containers of the ${target.attr('k8s.workload-type')} *${target.attr('steadybit.label')}* use the `:latest`-tag as version, which makes it hard to detect the actual deployed version in case of debugging.
<br/>
<br/>
**Affected Containers:** *<#list target.attrs('k8s.container.image.with-latest-tag') as item>${item}<#sep>, </#list>*
