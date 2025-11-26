// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package main

import (
	"context"
	"github.com/steadybit/extension-kubernetes/v2/ai"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/advice-kit/go/advice_kit_sdk"

	"runtime"
	"time"

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
	"github.com/steadybit/extension-kubernetes/v2/extreplicaset"
	"github.com/steadybit/extension-kubernetes/v2/extstatefulset"
	_ "go.uber.org/automaxprocs" // Importing automaxprocs automatically adjusts GOMAXPROCS.
)

func main() {
	stopCh := make(chan struct{})
	defer close(stopCh)
	extlogging.InitZeroLog()

	extconfig.ParseConfiguration()
	extconfig.ValidateConfiguration()
	//initKlogBridge(extconfig.Config.LogKubernetesHttpRequests)

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

		if client.K8S.Permissions().IsSetImageDeploymentPermitted() {
			action_kit_sdk.RegisterAction(extdeployment.NewSetImageAction())
		}
	}

	if !extconfig.Config.DiscoveryDisabledReplicaSet {
		discovery_kit_sdk.Register(extreplicaset.NewReplicaSetDiscovery(client.K8S))
		action_kit_sdk.RegisterAction(extreplicaset.NewReplicaSetPodCountCheckAction(client.K8S))
		if client.K8S.Permissions().IsScaleReplicaSetPermitted() {
			action_kit_sdk.RegisterAction(extreplicaset.NewScaleReplicaSetAction())
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

	if !extconfig.Config.DiscoveryDisabledIngress && client.K8S.Permissions().IsListIngressPermitted() && client.K8S.Permissions().IsListIngressClassesPermitted() && client.K8S.Permissions().IsModifyIngressPermitted() && !extconfig.HasNamespaceFilter() {
		discovery_kit_sdk.Register(extingress.NewIngressDiscovery(client.K8S))
		action_kit_sdk.RegisterAction(extingress.NewHAProxyBlockTrafficAction())
		action_kit_sdk.RegisterAction(extingress.NewHAProxyDelayTrafficAction())
		discovery_kit_sdk.Register(extingress.NewNginxIngressDiscovery(client.K8S))
		action_kit_sdk.RegisterAction(extingress.NewNginxBlockTrafficAction())
		action_kit_sdk.RegisterAction(extingress.NewNginxDelayTrafficAction())
	}

	if !extconfig.Config.DiscoveryDisabledNode && !extconfig.HasNamespaceFilter() {
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

	if extconfig.Config.EnableAIActions {
		bedrockClient, err := ai.NewAIClient(context.Background())
		if err != nil {
			panic(err)
		}
		action_kit_sdk.RegisterAction(ai.NewReliabilityCheckDeploymentAction(ai.ConverseWrapper{BedrockRuntimeClient: bedrockClient.BR}))
		action_kit_sdk.RegisterAction(ai.NewReliabilityCheckStatefulSetAction(ai.ConverseWrapper{BedrockRuntimeClient: bedrockClient.BR}))
		discovery_kit_sdk.Register(ai.NewReliabilityIssueDiscovery())
	}

	discovery_kit_sdk.Register(extcommon.NewAttributeDescriber())

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	adviceCfg := extconfig.Config.AdviceConfig
	advice_kit_sdk.RegisterAdvice(adviceCfg, deployment_strategy.GetAdviceDescriptionDeploymentStrategy)
	advice_kit_sdk.RegisterAdvice(adviceCfg, cpu_limit.GetAdviceDescriptionCPULimit)
	advice_kit_sdk.RegisterAdvice(adviceCfg, memory_limit.GetAdviceDescriptionMemoryLimit)
	advice_kit_sdk.RegisterAdvice(adviceCfg, ephemeral_storage_limit.GetAdviceDescriptionEphemeralStorageLimit)
	advice_kit_sdk.RegisterAdvice(adviceCfg, cpu_request.GetAdviceDescriptionCPURequest)
	advice_kit_sdk.RegisterAdvice(adviceCfg, memory_request.GetAdviceDescriptionMemoryRequest)
	advice_kit_sdk.RegisterAdvice(adviceCfg, ephemeral_storage_request.GetAdviceDescriptionEphemeralStorageRequest)
	advice_kit_sdk.RegisterAdvice(adviceCfg, image_latest_tag.GetAdviceDescriptionImageVersioning)
	advice_kit_sdk.RegisterAdvice(adviceCfg, image_pull_policy.GetAdviceDescriptionImagePullPolicy)
	advice_kit_sdk.RegisterAdvice(adviceCfg, probes.GetAdviceDescriptionProbes)
	advice_kit_sdk.RegisterAdvice(adviceCfg, single_replica.GetAdviceDescriptionSingleReplica)
	advice_kit_sdk.RegisterAdvice(adviceCfg, host_podantiaffinity.GetAdviceDescriptionHostPodantiaffinity)
	advice_kit_sdk.RegisterAdvice(adviceCfg, single_zone.GetAdviceDescriptionSingleZone)

	extsignals.ActivateSignalHandlers()
	action_kit_sdk.RegisterCoverageEndpoints()

	exthealth.SetReady(true)

	if extconfig.Config.PrintMemoryStatsInterval > 0 {
		ticker := time.NewTicker(time.Duration(extconfig.Config.PrintMemoryStatsInterval) * time.Second)
		go func() {
			for range ticker.C {
				client.K8S.PrintMemoryUsage()
				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				log.Info().
					Uint64("Alloc_kb", m.Alloc/1024).
					Uint64("TotalAlloc_kb", m.TotalAlloc/1024).
					Uint64("HeapAlloc_kb", m.HeapAlloc/1024).
					Uint64("HeapInUse_kb", m.HeapInuse/1024).
					Uint64("Sys_kb", m.Sys/1024).
					Uint32("NumGC", m.NumGC).
					Msg("Extension memory usage")
			}
		}()
	}

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
		AdviceList:    advice_kit_sdk.GetAdviceList(),
	}
}
