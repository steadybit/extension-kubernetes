// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extdeployment

import (
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kubernetes/advice"
)

const DeploymentStrategyID = DeploymentTargetType + ".advice.k8s-deployment-strategy"
const CpuLimitID = DeploymentTargetType + ".advice.k8s-cpu-limit"
const MemoryLimitID = DeploymentTargetType + ".advice.k8s-memory-limit"
const HorizontalPodAutoscalerID = DeploymentTargetType + ".advice.k8s-horizontal-pod-autoscaler"
const ImageVersioningID = DeploymentTargetType + ".advice.k8s-image-latest-tag"
const ImagePullPolicyID = DeploymentTargetType + ".advice.k8s-image-pull-policy"
const LivenessProbeID = DeploymentTargetType + ".advice.k8s-liveness-probe"
const ReadinessProbeID = DeploymentTargetType + ".advice.k8s-readiness-probe"
const SingleReplicaID = DeploymentTargetType + ".advice.k8s-single-replica"
const HostPodantiaffinityID = DeploymentTargetType + ".advice.k8s-host-podantiaffinity"

func RegisterDeploymentAdviceHandlers() {
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-deployment-strategy", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionDeploymentStrategy))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-cpu-limit", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionCPULimit))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-memory-limit", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionMemoryLimit))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-horizontal-pod-autoscaler", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionHorizontalPodAutoscaler))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-image-latest-tag", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionImageVersioning))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-image-pull-policy", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionImagePullPolicy))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-liveness-probe", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionLivenessProbe))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-readiness-probe", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionReadinessProbe))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-single-replica", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionSingleReplica))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-host-podantiaffinity", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionHostPodantiaffinity))
}

func getDeploymentAdviceDescriptionImageVersioning() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionImageVersioning(ImageVersioningID, DeploymentTargetType, "deployment")
}

func getDeploymentAdviceDescriptionImagePullPolicy() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionImagePullPolicy(ImagePullPolicyID, DeploymentTargetType, "deployment")
}

func getDeploymentAdviceDescriptionDeploymentStrategy() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionDeploymentStrategy(DeploymentStrategyID, DeploymentTargetType, "deployment")
}

func getDeploymentAdviceDescriptionHorizontalPodAutoscaler() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionHorizontalPodAutoscaler(HorizontalPodAutoscalerID, DeploymentTargetType, "deployment")
}

func getDeploymentAdviceDescriptionCPULimit() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionCPULimit(CpuLimitID, DeploymentTargetType, "deployment")
}
func getDeploymentAdviceDescriptionSingleReplica() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionSingleReplica(SingleReplicaID, DeploymentTargetType, "deployment")
}

func getDeploymentAdviceDescriptionHostPodantiaffinity() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionHostPodantiaffinity(HostPodantiaffinityID, DeploymentTargetType, "deployment")
}

func getDeploymentAdviceDescriptionLivenessProbe() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionLivenessProbe(LivenessProbeID, DeploymentTargetType, "deployment")
}

func getDeploymentAdviceDescriptionReadinessProbe() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionReadinessProbe(ReadinessProbeID, DeploymentTargetType, "deployment")
}

func getDeploymentAdviceDescriptionMemoryLimit() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionMemoryLimit(MemoryLimitID, DeploymentTargetType, "deployment")
}
