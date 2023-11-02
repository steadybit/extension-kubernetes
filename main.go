// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"github.com/rs/zerolog"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcluster"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/steadybit/extension-kubernetes/extcontainer"
	"github.com/steadybit/extension-kubernetes/extdaemonset"
	"github.com/steadybit/extension-kubernetes/extdeployment"
	"github.com/steadybit/extension-kubernetes/extevents"
	"github.com/steadybit/extension-kubernetes/extnode"
	"github.com/steadybit/extension-kubernetes/extpod"
	"github.com/steadybit/extension-kubernetes/extstatefulset"
	_ "net/http/pprof" //allow pprof
)

func main() {
	stopCh := make(chan struct{})
	defer close(stopCh)
	extlogging.InitZeroLog()

	extconfig.ParseConfiguration()
	extconfig.ValidateConfiguration()
	initKlogBridge(extconfig.Config.LogKubernetesHttpRequests)

	extbuild.PrintBuildInformation()
	extruntime.LogRuntimeInformation(zerolog.DebugLevel)

	exthealth.SetReady(false)
	exthealth.StartProbes(8089)

	client.PrepareClient(stopCh)

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	if client.K8S.Permissions().IsRolloutRestartPermitted() {
		action_kit_sdk.RegisterAction(extdeployment.NewDeploymentRolloutRestartAction())
	}
	action_kit_sdk.RegisterAction(extdeployment.NewCheckDeploymentRolloutStatusAction())
	action_kit_sdk.RegisterAction(extdeployment.NewPodCountCheckAction())
	action_kit_sdk.RegisterAction(extdeployment.NewPodCountMetricsAction())
	if client.K8S.Permissions().IsScaleDeploymentPermitted() {
		action_kit_sdk.RegisterAction(extdeployment.NewScaleDeploymentAction())
	}
	if client.K8S.Permissions().IsScaleStatefulSetPermitted() {
		action_kit_sdk.RegisterAction(extstatefulset.NewScaleStatefulSetAction())
	}
	if client.K8S.Permissions().IsDeletePodPermitted() {
		action_kit_sdk.RegisterAction(extpod.NewDeletePodAction())
	}
	if client.K8S.Permissions().IsCrashLoopPodPermitted() {
		action_kit_sdk.RegisterAction(extpod.NewCrashLoopAction())
	}
	action_kit_sdk.RegisterAction(extnode.NewNodeCountCheckAction())
	if client.K8S.Permissions().IsDrainNodePermitted() {
		action_kit_sdk.RegisterAction(extnode.NewDrainNodeAction())
	}
	if client.K8S.Permissions().IsTaintNodePermitted() {
		action_kit_sdk.RegisterAction(extnode.NewTaintNodeAction())
	}
	action_kit_sdk.RegisterAction(extevents.NewK8sEventsAction())

	extdeployment.RegisterAttributeDescriptionHandlers()
	extdeployment.RegisterDeploymentDiscoveryHandlers()
	extdeployment.RegisterDeploymentAdviceHandlers()
	extdaemonset.RegisterStatefulSetDiscoveryHandlers()
	extstatefulset.RegisterStatefulSetDiscoveryHandlers()
	extpod.RegisterPodDiscoveryHandlers()
	extcontainer.RegisterContainerDiscoveryHandlers()
	extnode.RegisterNodeDiscoveryHandlers()
	extcluster.RegisterClusterDiscoveryHandlers()

	action_kit_sdk.InstallSignalHandler()

	action_kit_sdk.RegisterCoverageEndpoints()

	exthealth.SetReady(true)

	exthttp.Listen(exthttp.ListenOpts{
		Port: 8088,
	})
}

type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
	advice_kit_api.AdviceList       `json:",inline"`
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		ActionList: action_kit_sdk.GetActionList(),
		DiscoveryList: discovery_kit_api.DiscoveryList{
			Discoveries: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/deployment/discovery",
				},
				{
					Method: "GET",
					Path:   "/statefulset/discovery",
				},
				{
					Method: "GET",
					Path:   "/daemonset/discovery",
				},
				{
					Method: "GET",
					Path:   "/pod/discovery",
				},
				{
					Method: "GET",
					Path:   "/container/discovery",
				},
				{
					Method: "GET",
					Path:   "/cluster/discovery",
				},
				{
					Method: "GET",
					Path:   "/node/discovery",
				},
			},
			TargetTypes: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/deployment/discovery/target-description",
				},
				{
					Method: "GET",
					Path:   "/statefulset/discovery/target-description",
				},
				{
					Method: "GET",
					Path:   "/daemonset/discovery/target-description",
				},
				{
					Method: "GET",
					Path:   "/pod//discovery/target-description",
				},
				{
					Method: "GET",
					Path:   "/container/discovery/target-description",
				},
				{
					Method: "GET",
					Path:   "/cluster/discovery/target-description",
				},
				{
					Method: "GET",
					Path:   "/node/discovery/target-description",
				},
			},
			TargetAttributes: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/attribute-descriptions",
				},
			},
			TargetEnrichmentRules: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/container/discovery/rules/k8s-container-to-container",
				},
				{
					Method: "GET",
					Path:   "/container/discovery/rules/k8s-container-to-host",
				},
				{
					Method: "GET",
					Path:   "/deployment/discovery/rules/k8s-deployment-to-container",
				},
				{
					Method: "GET",
					Path:   "/statefulset/discovery/rules/k8s-statefulset-to-container",
				},
				{
					Method: "GET",
					Path:   "/daemonset/discovery/rules/k8s-daemonset-to-container",
				},
				{
					Method: "GET",
					Path:   "/deployment/discovery/rules/container-to-k8s-deployment",
				},
				{
					Method: "GET",
					Path:   "/node/discovery/rules/k8s-node-to-host",
				},
			},
		},
		AdviceList: advice_kit_api.AdviceList{
			Advice: getAdviceRefs(),
		},
	}
}

func getAdviceRefs() []advice_kit_api.DescribingEndpointReference {
	var refs []advice_kit_api.DescribingEndpointReference
	refs = make([]advice_kit_api.DescribingEndpointReference, 0)
	for _, adviceId := range extconfig.Config.ActiveAdviceList {
		if adviceId == "*" || adviceId == extdeployment.DeploymentStrategyID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/deployment/advice/k8s-deployment-strategy",
			})
		}
		if adviceId == "*" || adviceId == extdeployment.CpuLimitID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/deployment/advice/k8s-cpu-limit",
			})
		}
		if adviceId == "*" || adviceId == extdeployment.MemoryLimitID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/deployment/advice/k8s-memory-limit",
			})
		}
		if adviceId == "*" || adviceId == extdeployment.HorizontalPodAutoscalerID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/deployment/advice/k8s-horizontal-pod-autoscaler",
			})
		}
		if adviceId == "*" || adviceId == extdeployment.ImageVersioningID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/deployment/advice/k8s-image-latest-tag",
			})
		}
		if adviceId == "*" || adviceId == extdeployment.ImagePullPolicyID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/deployment/advice/k8s-image-pull-policy",
			})
		}
		if adviceId == "*" || adviceId == extdeployment.LivenessProbeID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/deployment/advice/k8s-liveness-probe",
			})
		}
	}
	return refs
}
