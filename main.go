// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"context"
	_ "github.com/KimMachineGun/automemlimit" // By default, it sets `GOMEMLIMIT` to 90% of cgroup's memory limit.
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
	"github.com/steadybit/extension-kit/extsignals"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/cpu_limit"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/cpu_request"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/deployment_strategy"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/ephemeral_storage_limit"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/ephemeral_storage_request"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/host_podantiaffinity"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/image_latest_tag"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/image_pull_policy"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/memory_limit"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/memory_request"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/probes"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/single_replica"
	"github.com/steadybit/extension-kubernetes/v2/extadvice/single_zone"
	"github.com/steadybit/extension-kubernetes/v2/extcluster"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/steadybit/extension-kubernetes/v2/extcontainer"
	"github.com/steadybit/extension-kubernetes/v2/extdaemonset"
	"github.com/steadybit/extension-kubernetes/v2/extdeployment"
	"github.com/steadybit/extension-kubernetes/v2/extevents"
	"github.com/steadybit/extension-kubernetes/v2/extingress"
	"github.com/steadybit/extension-kubernetes/v2/extnode"
	"github.com/steadybit/extension-kubernetes/v2/extpod"
	"github.com/steadybit/extension-kubernetes/v2/extstatefulset"
	_ "go.uber.org/automaxprocs" // Importing automaxprocs automatically adjusts GOMAXPROCS.
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
		action_kit_sdk.RegisterAction(extdeployment.NewDeploymentPodCountCheckAction(client.K8S))
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
		action_kit_sdk.RegisterAction(extstatefulset.NewStatefulSetPodCountCheckAction(client.K8S))
		if client.K8S.Permissions().IsScaleStatefulSetPermitted() {
			action_kit_sdk.RegisterAction(extstatefulset.NewScaleStatefulSetAction())
		}
	}

	if !extconfig.Config.DiscoveryDisabledDaemonSet {
		discovery_kit_sdk.Register(extdaemonset.NewDaemonSetDiscovery(client.K8S))
		action_kit_sdk.RegisterAction(extdaemonset.NewDaemonSetPodCountCheckAction(client.K8S))
	}

	if !extconfig.Config.DiscoveryDisabledIngress && client.K8S.Permissions().IsModifyIngressAllowed() && !extconfig.IsUsingRoleBasedAccessControl(){
		discovery_kit_sdk.Register(extingress.NewIngressDiscovery(client.K8S))
		action_kit_sdk.RegisterAction(extingress.NewHAProxyBlockTrafficAction())
		action_kit_sdk.RegisterAction(extingress.NewHAProxyDelayTrafficAction())
	}

	if !extconfig.Config.DiscoveryDisabledNode && !extconfig.IsUsingRoleBasedAccessControl() {
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

	exthttp.RegisterHttpHandler("/advice/k8s-deployment-strategy", exthttp.GetterAsHandler(deployment_strategy.GetAdviceDescriptionDeploymentStrategy))
	exthttp.RegisterHttpHandler("/advice/k8s-cpu-limit", exthttp.GetterAsHandler(cpu_limit.GetAdviceDescriptionCPULimit))
	exthttp.RegisterHttpHandler("/advice/k8s-memory-limit", exthttp.GetterAsHandler(memory_limit.GetAdviceDescriptionMemoryLimit))
	exthttp.RegisterHttpHandler("/advice/k8s-ephemeral-storage-limit", exthttp.GetterAsHandler(ephemeral_storage_limit.GetAdviceDescriptionEphemeralStorageLimit))
	exthttp.RegisterHttpHandler("/advice/k8s-cpu-request", exthttp.GetterAsHandler(cpu_request.GetAdviceDescriptionCPURequest))
	exthttp.RegisterHttpHandler("/advice/k8s-memory-request", exthttp.GetterAsHandler(memory_request.GetAdviceDescriptionMemoryRequest))
	exthttp.RegisterHttpHandler("/advice/k8s-ephemeral-storage-request", exthttp.GetterAsHandler(ephemeral_storage_request.GetAdviceDescriptionEphemeralStorageRequest))
	exthttp.RegisterHttpHandler("/advice/k8s-image-latest-tag", exthttp.GetterAsHandler(image_latest_tag.GetAdviceDescriptionImageVersioning))
	exthttp.RegisterHttpHandler("/advice/k8s-image-pull-policy", exthttp.GetterAsHandler(image_pull_policy.GetAdviceDescriptionImagePullPolicy))
	exthttp.RegisterHttpHandler("/advice/k8s-probes", exthttp.GetterAsHandler(probes.GetAdviceDescriptionProbes))
	exthttp.RegisterHttpHandler("/advice/k8s-single-replica", exthttp.GetterAsHandler(single_replica.GetAdviceDescriptionSingleReplica))
	exthttp.RegisterHttpHandler("/advice/k8s-host-podantiaffinity", exthttp.GetterAsHandler(host_podantiaffinity.GetAdviceDescriptionHostPodantiaffinity))
	exthttp.RegisterHttpHandler("/advice/single-zone", exthttp.GetterAsHandler(single_zone.GetAdviceDescriptionSingleZone))

	extsignals.ActivateSignalHandlers()
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
		if adviceId == "*" || adviceId == deployment_strategy.DeploymentStrategyID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-deployment-strategy",
			})
		}
		if adviceId == "*" || adviceId == cpu_limit.CpuLimitID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-cpu-limit",
			})
		}
		if adviceId == "*" || adviceId == memory_limit.MemoryLimitID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-memory-limit",
			})
		}
		if adviceId == "*" || adviceId == ephemeral_storage_limit.EphemeralStorageLimitID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-ephemeral-storage-limit",
			})
		}
		if adviceId == "*" || adviceId == cpu_request.CpuRequestID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-cpu-request",
			})
		}
		if adviceId == "*" || adviceId == memory_request.MemoryRequestID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-memory-request",
			})
		}
		if adviceId == "*" || adviceId == ephemeral_storage_request.EphemeralStorageRequestID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-ephemeral-storage-request",
			})
		}
		if adviceId == "*" || adviceId == image_latest_tag.ImageVersioningID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-image-latest-tag",
			})
		}
		if adviceId == "*" || adviceId == image_pull_policy.ImagePullPolicyID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-image-pull-policy",
			})
		}
		if adviceId == "*" || adviceId == probes.ProbesID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-probes",
			})
		}
		if adviceId == "*" || adviceId == single_replica.SingleReplicaID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-single-replica",
			})
		}
		if adviceId == "*" || adviceId == host_podantiaffinity.HostPodantiaffinityID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/k8s-host-podantiaffinity",
			})
		}
		if adviceId == "*" || adviceId == single_zone.SingleZoneID {
			refs = append(refs, advice_kit_api.DescribingEndpointReference{
				Method: "GET",
				Path:   "/advice/single-zone",
			})
		}
	}
	return refs
}
