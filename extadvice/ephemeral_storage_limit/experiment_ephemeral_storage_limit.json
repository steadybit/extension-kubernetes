{
  "name": "Pod recovers after exceeding ephemeral storage of ${target.attr('steadybit.label')}",
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
          "type": "wait",
          "ignoreFailure": false,
          "parameters": {
            "duration": "30s"
          },
          "customLabel": "TODO VALIDATION: THEN: ${target.attr('steadybit.label')} e.g., still works, fails gracefully, scales up, or monitors alerts"
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
          "customLabel": "TODO VALIDATION: GIVEN: ${target.attr('steadybit.label')} works properly"
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
        },
        {
          "type": "wait",
          "ignoreFailure": false,
          "parameters": {
            "duration": "60s"
          },
          "customLabel": "TODO VALIDATION: ${target.attr('steadybit.label')} recovers from exceeding ephemeral storage and runs smoothly again"
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
