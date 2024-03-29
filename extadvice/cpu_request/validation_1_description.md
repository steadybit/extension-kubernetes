I *confirm* that the requested CPU resources are reasonable to avoid overcommitment of resources and optimize the scheduling of workload resources.

## What's the Risk?
Requesting a reasonable amount of CPU might be difficult and is always a tradeoff.
Setting the requested CPU too low may lead to starvation and the container not getting needed CPU cycles.
Setting the requested CPU too high may lead to inefficiency and extra CPUs.
In addition, the value may depend on the criticality of your Kubernetes resources.

## How to Identify Proper Values?
Historical data are a good indicator of what your Kubernetes resource needs. For instance, check the 99th percentile of the last 24 hours and add some extra headroom depending on the criticality of your Kubernetes resources.
If you don't have historical data, a load test may help you to generate more insights.
