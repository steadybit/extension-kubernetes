I *confirm* that the requested memory resources are reasonable to avoid overcommitment of resources and optimize the scheduling of workload resources.

## What's the Risk?
Requesting a reasonable amount of memory might be difficult and is always a tradeoff.
Setting the requested memory too low may lead to Kubernetes killing the container.
Setting the requested memory too high may lead to inefficiency and extra memory.
In addition, the value may depend on the criticality of your Kubernetes resources.

## How to Identify Proper Values?
Historical data are a good indicator of what your Kubernetes resource needs. For instance, check the 99th percentile of the last 24 hours and add some extra headroom depending on the criticality of your Kubernetes resources.
If you don't have historical data, a load test may help you to generate more insights.
