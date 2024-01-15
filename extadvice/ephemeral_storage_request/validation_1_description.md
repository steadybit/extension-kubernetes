I *confirm* that the requested ephemeral storage resources are reasonable to avoid overcommitment of resources and optimize the scheduling of workload resources.

## What's the Risk?
Requesting a reasonable amount of ephemeral storage might be difficult and is always a tradeoff.
Setting the requested ephemeral storage too low may lead to running out of disk when writing e.g. temporary files and thus performance issues.
Setting the requested ephemeral storage too high may lead to inefficiency and extra disk space.

## How to Identify Proper Values?
Historical data are a good indicator of what your Kubernetes resource needs.
If you don't have historical data, a load test may help you to generate more insights.
