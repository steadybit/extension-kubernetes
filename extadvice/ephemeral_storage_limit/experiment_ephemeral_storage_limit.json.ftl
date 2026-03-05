{
	"templateTitle": "Ephemeral Storage Overload",
	"templateDescription": "Check what happens when exceeding ephemeral storage. Are all Kubernetes resources working properly, failing gracefully, or raising a monitor alert?",
	"placeholders": [
		<#if target.attr('service.id')?? && target.attr('service.id') != '<unknown>'>
		<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='http'>
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
	"experimentName": "Pod recovers after exceeding ephemeral storage of ${target.attr('steadybit.label')}",
  "lanes": [
    {
			"steps": [
				{
					"type": "wait",
					"ignoreFailure": false,
					"parameters": {
						"duration": "150s"
					},
					"customLabel": "TODO: Consider updating the megabytes written to the disk to exceed the configured ephemeral storage of ${target.attr('steadybit.label')}"
				}
			]
		},
		{
      "steps": [
				<#if target.attr('service.id')?? && target.attr('service.id') != '<unknown>'>
					{
						"type": "service-validation",
						"ignoreFailure": false,
						"parameters": {
							"duration": "150s"
						},
						"serviceId": "${target.attr('service.id', 0)}",
				  	"customLabel": "INVARIANT: ${target.attr('steadybit.label')}'s features work within expected success rates"
					}
				<#elseif target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='http'>
					{
					"type": "action",
					"ignoreFailure": false,
					"parameters": {
					"duration": "150s",
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
            "duration": "10s"
          },
					"customLabel": "TODO VALIDATION: GIVEN: ${target.attr('steadybit.label')} works properly"
        },
				{
					"type": "wait",
					"ignoreFailure": false,
					"parameters": {
						"duration": "30s"
					},
					"customLabel": "TODO VALIDATION: THEN: ${target.attr('steadybit.label')} e.g., still works, fails gracefully, scales up, or monitors alerts"
				},
				{
					"type": "wait",
					"ignoreFailure": false,
					"parameters": {
						"duration": "60s"
					},
					"customLabel": "TODO VALIDATION: ${target.attr('steadybit.label')} recovers from exceeding ephemeral storage and runs smoothly again"
				}
				</#if>
      ]
    },
    {
      "steps": [
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "duration": "10s",
            "podCountCheckMode": "podCountEqualsDesiredCount"
          },
          "customLabel": "GIVEN: All pods are ready",
          "actionType": "<#if target.id.type=='com.steadybit.extension_kubernetes.kubernetes-deployment'>com.steadybit.extension_kubernetes.pod_count_check<#elseif target.id.type=='com.steadybit.extension_kubernetes.kubernetes-statefulset'>com.steadybit.extension_kubernetes.pod_count_check_statefulset<#else>com.steadybit.extension_kubernetes.pod_count_check_daemonset</#if>",
          "radius": {
            "targetType": "${target.id.type}",
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
            "query": null
          }
        }
      ]
    },
    {
      "steps": [
        {
          "type": "wait",
          "ignoreFailure": false,
          "parameters": {
            "duration": "10s"
          },
					"customLabel": "Wait for fill disk"
        },
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "mode": "MB_TO_FILL",
            "path": "/tmp",
            "size": 2000,
            "method": "AT_ONCE",
            "duration": "30s",
            "blocksize": 5
          },
          "customLabel": "WHEN: ${target.attr('steadybit.label')}'s pod exceeds ephemeral storage",
          "actionType": "com.steadybit.extension_container.fill_disk",
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
          "type": "wait",
          "ignoreFailure": false,
          "parameters": {
            "duration": "10s"
          },
          "customLabel": "Wait for fill disk"
        },
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "duration": "20s",
            "podCountCheckMode": "podCountLessThanDesiredCount"
          },
          "customLabel": "THEN: Kubernetes restarts one pod due to exceeding ephemeral storage",
          "actionType": "<#if target.id.type=='com.steadybit.extension_kubernetes.kubernetes-deployment'>com.steadybit.extension_kubernetes.pod_count_check<#elseif target.id.type=='com.steadybit.extension_kubernetes.kubernetes-statefulset'>com.steadybit.extension_kubernetes.pod_count_check_statefulset<#else>com.steadybit.extension_kubernetes.pod_count_check_daemonset</#if>",
          "radius": {
            "targetType": "${target.id.type}",
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
            "query": null
          }
        },
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "duration": "60s",
            "podCountCheckMode": "podCountEqualsDesiredCount"
          },
          "customLabel": "THEN: Pod recovers within 60 seconds",
          "actionType": "<#if target.id.type=='com.steadybit.extension_kubernetes.kubernetes-deployment'>com.steadybit.extension_kubernetes.pod_count_check<#elseif target.id.type=='com.steadybit.extension_kubernetes.kubernetes-statefulset'>com.steadybit.extension_kubernetes.pod_count_check_statefulset<#else>com.steadybit.extension_kubernetes.pod_count_check_daemonset</#if>",
          "radius": {
            "targetType": "${target.id.type}",
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
            "duration": "150s"
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
            "duration": "150s"
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
