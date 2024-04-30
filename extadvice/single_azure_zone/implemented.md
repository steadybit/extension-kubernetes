Right now, when availability zone *${target.attr('azure.zone',0)}* fails, your service *${target.attr('steadybit.label')}* will still be available because you use *${target.attrs('azure.zone')?size}* zones to handle requests.

