An availability zone can be unavailable as they are not redundantly designed.
In order to survive an outage of the availability zone *${target.attr('gcp.zone',0)}* you should spread your Kubernetes pods across multiple availability zones.
