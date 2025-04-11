package extpod

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
)

const (
	PodTargetType     = "com.steadybit.extension_kubernetes.kubernetes-pod"
	DeletePodActionId = "com.steadybit.extension_kubernetes.delete_pod"
	CrashLoopActionId = "com.steadybit.extension_kubernetes.crash_loop_pod"
)

var (
	targetSelectionTemplates = action_kit_api.TargetSelection{
		TargetType: PodTargetType,
		SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "deployment",
				Description: extutil.Ptr("Find pods by cluster, namespace and deployment"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
			},
			{
				Label:       "statefulset",
				Description: extutil.Ptr("Find pods by cluster, namespace and statefulset."),
				Query:       "k8s.cluster-name=\"\" and k8s.namespace=\"\" and k8s.statefulset=\"\"",
			},
			{
				Label:       "daemonset",
				Description: extutil.Ptr("Find pods by cluster, namespace and daemonset."),
				Query:       "k8s.cluster-name=\"\" and k8s.namespace=\"\" and k8s.daemonset=\"\"",
			},
		}),
	}
)
