{
  "templateTitle": "Unhealthiness of a pod's container is detected",
  "templateDescription": "Check what happens when a container of a pod isn't healthy. How long does it take to restart the container? What happens with the provided service in the meantime?",
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
  "tags": ["Advice", "Redundancy"],
  "experimentName": "Unhealthiness of ${target.attr('steadybit.label')} is detected",
  "hypothesis": "When a container of ${target.attr('steadybit.label')} isn't healthy, Kubernetes will restart the container and routes traffic as soon as it is healthy again.",
  "lanes": [
    {
      "steps": [
        <#if target.attr('k8s.label.tags.steadybit.com/service-validation')?? && target.attr('k8s.label.tags.steadybit.com/service-validation')=='http'>
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "duration": "140s",
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
          "radius": {}
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
          "radius": {}
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
          "radius": {}
        }
        <#else>
        {
          "type": "wait",
          "ignoreFailure": false,
          "parameters": {
            "duration": "140s"
          },
          "customLabel": "TODO VALIDATION: INVARIANT: ${target.attr('steadybit.label')}'s features work within expected success rates"
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
            "duration": "20s",
            "podCountCheckMode": "podCountEqualsDesiredCount"
          },
          "customLabel": "GIVEN: All pods are healthy",
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
            "duration": "20s"
          }
        },
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "ip": [],
            "port": [],
            "duration": "60s",
            "hostname": [],
            "failOnHostNetwork": true
          },
          "customLabel": "WHEN: One container's pod is unhealthy",
          "actionType": "com.steadybit.extension_container.network_blackhole",
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
            "percentage": 50
          }
        },
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "duration": "60s",
            "podCountCheckMode": "podCountEqualsDesiredCount"
          },
          "customLabel": "THEN: All pods are healthy again within 60s",
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
            "duration": "80s",
            "podCountCheckMode": "podCountLessThanDesiredCount"
          },
          "customLabel": "THEN: One pod is detected unhealthy and restarted",
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
            "duration": "140s"
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
            "duration": "140s"
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
