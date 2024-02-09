Some containers of the ${target.k8s.workload-type:normal} ${target.steadybit.label} use the `:latest`-tag as version, which makes it hard to detect the actual deployed version in case of debugging.
<br/>
<br/>
**Affected Containers:** ${target.k8s.container.image.with-latest-tag[]}

