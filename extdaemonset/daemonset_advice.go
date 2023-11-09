// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extdaemonset

import (
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kubernetes/advice"
)

const CpuLimitID = DaemonSetTargetType + ".advice.k8s-cpu-limit"
const MemoryLimitID = DaemonSetTargetType + ".advice.k8s-memory-limit"
const ImageVersioningID = DaemonSetTargetType + ".advice.k8s-image-latest-tag"
const ImagePullPolicyID = DaemonSetTargetType + ".advice.k8s-image-pull-policy"
const LivenessProbeID = DaemonSetTargetType + ".advice.k8s-liveness-probe"
const ReadinessProbeID = DaemonSetTargetType + ".advice.k8s-readiness-probe"
const SingleAwsZoneID = DaemonSetTargetType + ".advice.single-aws-zone"
func RegisterDaemonsetAdviceHandlers() {
	exthttp.RegisterHttpHandler("/daemonset/advice/k8s-cpu-limit", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionCPULimit))
	exthttp.RegisterHttpHandler("/daemonset/advice/k8s-memory-limit", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionMemoryLimit))
	exthttp.RegisterHttpHandler("/daemonset/advice/k8s-image-latest-tag", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionImageVersioning))
	exthttp.RegisterHttpHandler("/daemonset/advice/k8s-image-pull-policy", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionImagePullPolicy))
	exthttp.RegisterHttpHandler("/daemonset/advice/k8s-liveness-probe", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionLivenessProbe))
	exthttp.RegisterHttpHandler("/daemonset/advice/k8s-readiness-probe", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionReadinessProbe))
	exthttp.RegisterHttpHandler("/daemonset/advice/single-aws-zone", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionSingleAwsZone))
}

func getDeploymentAdviceDescriptionImageVersioning() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionImageVersioning(ImageVersioningID, DaemonSetTargetType, "daemonset")
}

func getDeploymentAdviceDescriptionImagePullPolicy() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionImagePullPolicy(ImagePullPolicyID, DaemonSetTargetType, "daemonset")
}

func getDeploymentAdviceDescriptionCPULimit() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionCPULimit(CpuLimitID, DaemonSetTargetType, "daemonset")
}

func getDeploymentAdviceDescriptionMemoryLimit() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionMemoryLimit(MemoryLimitID, DaemonSetTargetType, "daemonset")
}

func getDeploymentAdviceDescriptionLivenessProbe() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionLivenessProbe(LivenessProbeID, DaemonSetTargetType, "daemonset")
}

func getDeploymentAdviceDescriptionReadinessProbe() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionReadinessProbe(ReadinessProbeID, DaemonSetTargetType, "daemonset")
}

func getDeploymentAdviceDescriptionSingleAwsZone() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionReadinessProbe(SingleAwsZoneID, DaemonSetTargetType, "daemonset")
}



