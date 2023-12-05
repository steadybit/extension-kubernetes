// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extadvice"
	"github.com/steadybit/extension-kubernetes/extcluster"
	"github.com/steadybit/extension-kubernetes/extcommon"
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

	if !extconfig.Config.DiscoveryDisabledDeployment {
		discovery_kit_sdk.Register(extdeployment.NewDeploymentDiscovery(client.K8S))
		action_kit_sdk.RegisterAction(extdeployment.NewCheckDeploymentRolloutStatusAction())
		action_kit_sdk.RegisterAction(extdeployment.NewPodCountCheckAction())
		if client.K8S.Permissions().IsRolloutRestartPermitted() {
			action_kit_sdk.RegisterAction(extdeployment.NewDeploymentRolloutRestartAction())
		}
		if client.K8S.Permissions().IsScaleDeploymentPermitted() {
			action_kit_sdk.RegisterAction(extdeployment.NewScaleDeploymentAction())
		}
	}

	if !extconfig.Config.DiscoveryDisabledPod {
		discovery_kit_sdk.Register(extpod.NewPodDiscovery(client.K8S))
		if client.K8S.Permissions().IsDeletePodPermitted() {
			action_kit_sdk.RegisterAction(extpod.NewDeletePodAction())
		}
		if client.K8S.Permissions().IsCrashLoopPodPermitted() {
			action_kit_sdk.RegisterAction(extpod.NewCrashLoopAction())
		}
	}

	if !extconfig.Config.DiscoveryDisabledStatefulSet {
		discovery_kit_sdk.Register(extstatefulset.NewStatefulSetDiscovery(client.K8S))
		if client.K8S.Permissions().IsScaleStatefulSetPermitted() {
			action_kit_sdk.RegisterAction(extstatefulset.NewScaleStatefulSetAction())
		}
	}

	if !extconfig.Config.DiscoveryDisabledDaemonSet {
		discovery_kit_sdk.Register(extdaemonset.NewDaemonSetDiscovery(client.K8S))
	}

	if !extconfig.Config.DiscoveryDisabledNode {
		discovery_kit_sdk.Register(extnode.NewNodeDiscovery(client.K8S))
		action_kit_sdk.RegisterAction(extnode.NewNodeCountCheckAction())

		if client.K8S.Permissions().IsDrainNodePermitted() {
			action_kit_sdk.RegisterAction(extnode.NewDrainNodeAction())
		}
		if client.K8S.Permissions().IsTaintNodePermitted() {
			action_kit_sdk.RegisterAction(extnode.NewTaintNodeAction())
		}
	}

	if !extconfig.Config.DiscoveryDisabledContainer {
		discovery_kit_sdk.Register(extcontainer.NewContainerDiscovery(context.Background(), client.K8S))
	}

	if !extconfig.Config.DiscoveryDisabledCluster {
		discovery_kit_sdk.Register(extcluster.NewClusterDiscovery())
		action_kit_sdk.RegisterAction(extdeployment.NewPodCountMetricsAction())
		action_kit_sdk.RegisterAction(extevents.NewK8sEventsAction())
	}

	discovery_kit_sdk.Register(extcommon.NewAttributeDescriber())

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	extadvice.RegisterAdviceHandlers()

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
		ActionList:    action_kit_sdk.GetActionList(),
		DiscoveryList: discovery_kit_sdk.GetDiscoveryList(),
		AdviceList: advice_kit_api.AdviceList{
			Advice: getAdviceRefs(),
		},
	}
}

func getAdviceRefs() []advice_kit_api.DescribingEndpointReference {
	var refs []advice_kit_api.DescribingEndpointReference
	refs = make([]advice_kit_api.DescribingEndpointReference, 0)
	for _, adviceId := range extconfig.Config.ActiveAdviceList {
		// Deployments
		if adviceId == "*" || adviceId == extadvice.DeploymentStrategyID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-deployment-strategy",
			})
		}
		if adviceId == "*" || adviceId == extadvice.CpuLimitID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-cpu-limit",
			})
		}
		if adviceId == "*" || adviceId == extadvice.MemoryLimitID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-memory-limit",
			})
		}
		if adviceId == "*" || adviceId == extadvice.CpuRequestID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-cpu-request",
			})
		}
		if adviceId == "*" || adviceId == extadvice.MemoryRequestID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-memory-request",
			})
		}
		if adviceId == "*" || adviceId == extadvice.HorizontalPodAutoscalerID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-horizontal-pod-autoscaler",
			})
		}
		if adviceId == "*" || adviceId == extadvice.ImageVersioningID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-image-latest-tag",
			})
		}
		if adviceId == "*" || adviceId == extadvice.ImagePullPolicyID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-image-pull-policy",
			})
		}
		if adviceId == "*" || adviceId == extadvice.LivenessProbeID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-liveness-probe",
			})
		}
		if adviceId == "*" || adviceId == extadvice.ReadinessProbeID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-readiness-probe",
			})
		}
		if adviceId == "*" || adviceId == extadvice.SingleReplicaID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-single-replica",
			})
		}
		if adviceId == "*" || adviceId == extadvice.HostPodantiaffinityID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-host-podantiaffinity",
			})
		}
		if adviceId == "*" || adviceId == extadvice.SingleAWSZoneID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/single-aws-zone",
			})
		}
		if adviceId == "*" || adviceId == extadvice.SingleAzureZoneID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/single-azure-zone",
			})
		}
	}
	return refs
}
