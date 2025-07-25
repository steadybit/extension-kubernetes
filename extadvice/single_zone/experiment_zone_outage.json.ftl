{
  "templateTitle": "Zone outage",
  "templateDescription": "Check what happens when a zone is down and validate that Kubernetes manages this accordingly by routing the traffic within expected failure rates so that the offered features still work. As soon as the zone is available again, the pod should be ready again within 60s.",
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
  "tags": ["Redundancy", "Cloud", "Availability Zone", "Advice"],
  "experimentName": "Zone Outage of ${target.attr('k8s.label.topology.kubernetes.io/zone', 0)} for ${target.attr('steadybit.label')}",
  "hypothesis": "When Zone ${target.attr('k8s.label.topology.kubernetes.io/zone', 0)} is down for ${target.attr('steadybit.label')}, Kubernetes manages this accordingly by routing the traffic within expected failure rates so that the offered features still work. As soon as the zone is available again, the pod is ready within 60s.",
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
          "customLabel": "GIVEN: All pods in ${target.attr('k8s.label.topology.kubernetes.io/zone', 0)} are ready",
          "actionType": "<#if target.id.type=='com.steadybit.extension_kubernetes.kubernetes-deployment'>com.steadybit.extension_kubernetes.pod_count_check<#elseif target.id.type=='com.steadybit.extension_kubernetes.kubernetes-statefulset'>com.steadybit.extension_kubernetes.pod_count_check_statefulset<#else>com.steadybit.extension_kubernetes.pod_count_check_daemonset</#if>",
          "radius": {
            "targetType": "${target.id.type}",
            "predicate": {
              "operator": "AND",
              "predicates": [
                {
                  "key": "k8s.label.topology.kubernetes.io/zone",
                  "operator": "EQUALS",
                  "values": [
                    "${target.attr('k8s.label.topology.kubernetes.io/zone', 0)}"
                  ]
                },
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
          },
          "customLabel": "Wait for Zone outage"
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
          "customLabel": "WHEN: Zone outage of ${target.attr('k8s.label.topology.kubernetes.io/zone', 0)} for ${target.attr('steadybit.label')}",
          "actionType": "com.steadybit.extension_container.network_blackhole",
          "radius": {
            "targetType": "com.steadybit.extension_container.container",
            "predicate": {
              "operator": "AND",
              "predicates": [
                {
                  "key": "k8s.label.topology.kubernetes.io/zone",
                  "operator": "EQUALS",
                  "values": [
                    "${target.attr('k8s.label.topology.kubernetes.io/zone', 0)}"
                  ]
                },
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
            "percentage": 100
          }
        },
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "duration": "60s",
            "podCountCheckMode": "podCountEqualsDesiredCount"
          },
          "customLabel": "THEN: After Zone outage, all pods become ready again within 60s",
          "actionType": "<#if target.id.type=='com.steadybit.extension_kubernetes.kubernetes-deployment'>com.steadybit.extension_kubernetes.pod_count_check<#elseif target.id.type=='com.steadybit.extension_kubernetes.kubernetes-statefulset'>com.steadybit.extension_kubernetes.pod_count_check_statefulset<#else>com.steadybit.extension_kubernetes.pod_count_check_daemonset</#if>",
          "radius": {
            "targetType": "${target.id.type}",
            "predicate": {
              "operator": "AND",
              "predicates": [
                {
                  "key": "k8s.label.topology.kubernetes.io/zone",
                  "operator": "EQUALS",
                  "values": [
                    "${target.attr('k8s.label.topology.kubernetes.io/zone', 0)}"
                  ]
                },
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
          },
          "customLabel": "Wait for Zone outage"
        },
        {
          "type": "action",
          "ignoreFailure": false,
          "parameters": {
            "duration": "45s",
            "podCountCheckMode": "podCountLessThanDesiredCount"
          },
          "customLabel": "THEN: Pods are detected as down",
          "actionType": "<#if target.id.type=='com.steadybit.extension_kubernetes.kubernetes-deployment'>com.steadybit.extension_kubernetes.pod_count_check<#elseif target.id.type=='com.steadybit.extension_kubernetes.kubernetes-statefulset'>com.steadybit.extension_kubernetes.pod_count_check_statefulset<#else>com.steadybit.extension_kubernetes.pod_count_check_daemonset</#if>",
          "radius": {
            "targetType": "${target.id.type}",
            "predicate": {
              "operator": "AND",
              "predicates": [
                {
                  "key": "k8s.label.topology.kubernetes.io/zone",
                  "operator": "EQUALS",
                  "values": [
                    "${target.attr('k8s.label.topology.kubernetes.io/zone', 0)}"
                  ]
                },
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
