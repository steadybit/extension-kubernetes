{
	"templateTitle": "CPU Overload",
	"templateDescription": "Check what happens when stressing CPU resources. Are all Kubernetes resources working properly, failing gracefully, scaling up, or raising a monitor alert?",
	"placeholders": [
		<#if target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='http'>
		{
			"key": "httpLoadBalancedEndpoint",
			"name": "HTTP Load Balanced Endpoint",
			"description": "### What is the URL of a **load-balanced HTTP endpoint** served by the Kubernetes workload?\n\nWe will use the HTTP endpoint to validate that the provided service's features are working fine for the entire experiment duration."
		}
		<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='k6'>
		{
			"key": "k6LoadTestFile",
			"name": "k6 Load Test File",
			"description": "### Specify a k6 load test file to validate the service's functionality.\n\nWe will use the load test to validate that the provided service's features are working fine for the entire experiment duration."
		}
		<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='jmeter'>
		{
			"key": "jmeterLoadTestFile",
			"name": "JMeter Load Test File",
			"description": "### Specify a JMeter load test file to validate the service's functionality.\n\nWe will use the load test to validate that the provided service's features are working fine for the entire experiment duration."
		}
		<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='gatling'>
		{
			"key": "gatlingLoadTestFile",
			"name": "Gatling Load Test File",
			"description": "### Specify a Gatling load test file to validate the service's functionality.\n\nWe will use the load test to validate that the provided service's features are working fine for the entire experiment duration."
		}
		</#if>
	],
	"tags": ["Advice", "Resources"],
	"experimentName": "CPU Overload of ${target.attr('steadybit.label')}",
	"hypothesis": "In case of a CPU overload of ${target.attr('steadybit.label')}, all Kubernetes resources work properly, fail gracefully, scale up, or at least one monitor alert.",
	"lanes": [
		{
			"steps": [
				<#if target.attr('service.id')?? && target.attr('service.id') != '<unknown>'>
					{
						"type": "service-validation",
						"ignoreFailure": false,
						"parameters": {
						"duration": "160s"
						},
						"serviceId": "${target.attr('service.id')}",
						"customLabel": "INVARIANT: ${target.attr('steadybit.label')}'s features work within expected success rates"
					}
				<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='http'>
					{
						"type": "action",
						"ignoreFailure": false,
						"parameters": {
						"duration": "160s",
						"headers": [],
						"method": "GET",
						"successRate": 100,
						"maxConcurrent": 5,
						"followRedirects": false,
						"readTimeout": "5s",
						"connectTimeout": "5s",
						"requestsPerSecond": 10,
						"url": "[[httpLoadBalancedEndpoint]]",
						"statusCode": "200-299"
						},
						"customLabel": "INVARIANT: ${target.attr('steadybit.label')}'s features work within expected success rates",
						"actionType": "com.steadybit.extension_http.check.periodically",
						"radius": {}
					}
				<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='k6'>
					{
						"type": "action",
						"ignoreFailure": false,
						"parameters": {
						"environment": [],
							"file": "[[k6LoadTestFile]]"
						},
						"actionType": "com.steadybit.extension_k6.run",
						"radius": {},
						"customLabel": "INVARIANT: ${target.attr('steadybit.label')}'s features work within expected success rates"
					}
				<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='jmeter'>
					{
						"type": "action",
						"ignoreFailure": false,
						"parameters": {
							"file": "[[jmeterLoadTestFile]]",
						"parameter": []
						},
						"actionType": "com.steadybit.extension_jmeter.run",
						"radius": {},
						"customLabel": "INVARIANT: ${target.attr('steadybit.label')}'s features work within expected success rates"
					}
				<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='gatling'>
					{
						"type": "action",
						"ignoreFailure": false,
						"parameters": {
							"file": "[[gatlingLoadTestFile]]",
						"parameter": []
						},
						"actionType": "com.steadybit.extension_gatling.run",
						"radius": {},
						"customLabel": "INVARIANT: ${target.attr('steadybit.label')}'s features work within expected success rates"
					}
				<#else>
					{
						"type": "wait",
						"ignoreFailure": false,
						"parameters": {
							"duration": "40s"
						},
						"customLabel": "TODO VALIDATION: GIVEN: ${target.attr('steadybit.label')} works properly"
					},
					{
						"type": "wait",
						"ignoreFailure": false,
						"parameters": {
							"duration": "60s"
						},
						"customLabel": "TODO VALIDATION: THEN: ${target.attr('steadybit.label')} e.g., still works, fails gracefully, scales up, or monitors alerts"
					},
					{
						"type": "wait",
						"ignoreFailure": false,
						"parameters": {
							"duration": "60s"
						},
						"customLabel": "TODO VALIDATION: THEN: ${target.attr('steadybit.label')} recovers from high CPU and runs smoothly again"
					}
				</#if>
			]
		},
		{
			"steps": [
				{
					"type": "wait",
					"ignoreFailure": false,
					"parameters": {
						"duration": "40s"
					},
					"customLabel": "Wait for CPU stress"
				},
				{
					"type": "action",
					"ignoreFailure": false,
					"parameters": {
						"cpuLoad": 100,
						"workers": 0,
						"duration": "60s"
					},
					"customLabel": "WHEN: Stressing CPU of ${target.attr('steadybit.label')}",
					"actionType": "com.steadybit.extension_container.stress_cpu",
					"radius": {
						"targetType": "com.steadybit.extension_container.container",
						"predicate": {
							"operator": "AND",
							"predicates": [
								{
									"key": "k8s.cluster-name",
									"operator": "EQUALS",
									"values": [
										"${target.attr('k8s.cluster-name')}"
									]
								},
								{
									"key": "k8s.namespace",
									"operator": "EQUALS",
									"values": [
										"${target.attr('k8s.namespace')}"
									]
								},
								{
									"key": "k8s.${target.attr('k8s.workload-type')}",
									"operator": "EQUALS",
									"values": [
										"${target.attr('k8s.workload-owner')}"
									]
								}
							]
						},
						"query": null,
						"maximum": 1
					}
				}
			]
		},
		{
			"steps": [
				{
					"type": "action",
					"ignoreFailure": false,
					"parameters": {
						"duration": "160s"
					},
					"customLabel": "Show Kubernetes events from the cluster",
					"actionType": "com.steadybit.extension_kubernetes.kubernetes_logs",
					"radius": {
						"targetType": "com.steadybit.extension_kubernetes.kubernetes-cluster",
						"predicate": {
							"operator": "AND",
							"predicates": [
								{
									"key": "k8s.cluster-name",
									"operator": "EQUALS",
									"values": [
										"${target.attr('k8s.cluster-name')}"
									]
								}
							]
						},
						"query": null
					}
				}
			]
		},
		{
			"steps": [
				{
					"type": "action",
					"ignoreFailure": false,
					"parameters": {
						"duration": "160s"
					},
					"customLabel": "Show Pod Count Metrics for the cluster",
					"actionType": "com.steadybit.extension_kubernetes.pod_count_metric",
					"radius": {
						"targetType": "com.steadybit.extension_kubernetes.kubernetes-cluster",
						"predicate": {
							"operator": "AND",
							"predicates": [
								{
									"key": "k8s.cluster-name",
									"operator": "EQUALS",
									"values": [
										"${target.attr('k8s.cluster-name')}"
									]
								}
							]
						},
						"query": null
					}
				}
			]
		}
	]
}
