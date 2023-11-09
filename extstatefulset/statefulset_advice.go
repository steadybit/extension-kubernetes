// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extstatefulset

import (
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kubernetes/advice"
)

const CpuLimitID = StatefulSetTargetType + ".advice.k8s-cpu-limit"
const MemoryLimitID = StatefulSetTargetType + ".advice.k8s-memory-limit"
const HorizontalPodAutoscalerID = StatefulSetTargetType + ".advice.k8s-horizontal-pod-autoscaler"
const ImageVersioningID = StatefulSetTargetType + ".advice.k8s-image-latest-tag"
const ImagePullPolicyID = StatefulSetTargetType + ".advice.k8s-image-pull-policy"
const LivenessProbeID = StatefulSetTargetType + ".advice.k8s-liveness-probe"
const ReadinessProbeID = StatefulSetTargetType + ".advice.k8s-readiness-probe"
const SingleReplicaID = StatefulSetTargetType + ".advice.k8s-single-replica"
const HostPodantiaffinityID = StatefulSetTargetType + ".advice.k8s-host-podantiaffinity"
const SingleAWSZoneID = StatefulSetTargetType + ".advice.single-aws-zone"
const SingleAzureZoneID = StatefulSetTargetType + ".advice.single-azure-zone"

func RegisterStatefulsetAdviceHandlers() {
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-cpu-limit", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionCPULimit))
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-memory-limit", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionMemoryLimit))
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-horizontal-pod-autoscaler", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionHorizontalPodAutoscaler))
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-image-latest-tag", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionImageVersioning))
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-image-pull-policy", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionImagePullPolicy))
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-liveness-probe", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionLivenessProbe))
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-readiness-probe", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionReadinessProbe))
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-single-replica", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionSingleReplica))
	exthttp.RegisterHttpHandler("/statefulset/advice/k8s-host-podantiaffinity", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionHostPodantiaffinity))
	exthttp.RegisterHttpHandler("/statefulset/advice/single-aws-zone", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionSingleAwsZone))
	exthttp.RegisterHttpHandler("/statefulset/advice/single-azure-zone", exthttp.GetterAsHandler(getStatefulsetAdviceDescriptionSingleAzureZone))
}

func getStatefulsetAdviceDescriptionImageVersioning() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionImageVersioning(ImageVersioningID, StatefulSetTargetType, "statefulset")
}

func getStatefulsetAdviceDescriptionImagePullPolicy() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionImagePullPolicy(ImagePullPolicyID, StatefulSetTargetType, "statefulset")
}

func getStatefulsetAdviceDescriptionHorizontalPodAutoscaler() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionHorizontalPodAutoscaler(HorizontalPodAutoscalerID, StatefulSetTargetType, "statefulset")
}

func getStatefulsetAdviceDescriptionCPULimit() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionCPULimit(CpuLimitID, StatefulSetTargetType, "statefulset")
}

func getStatefulsetAdviceDescriptionMemoryLimit() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionMemoryLimit(MemoryLimitID, StatefulSetTargetType, "statefulset")
}

func getStatefulsetAdviceDescriptionLivenessProbe() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionLivenessProbe(LivenessProbeID, StatefulSetTargetType, "statefulset")
}

func getStatefulsetAdviceDescriptionReadinessProbe() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionReadinessProbe(ReadinessProbeID, StatefulSetTargetType, "statefulset")
}
func getStatefulsetAdviceDescriptionSingleReplica() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionSingleReplica(SingleReplicaID, StatefulSetTargetType, "statefulset")
}

func getStatefulsetAdviceDescriptionHostPodantiaffinity() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionHostPodantiaffinity(HostPodantiaffinityID, StatefulSetTargetType, "statefulset")
}
func getStatefulsetAdviceDescriptionSingleAwsZone() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionSingleAwsZone(SingleAWSZoneID, StatefulSetTargetType, "statefulset")
}

func getStatefulsetAdviceDescriptionSingleAzureZone() advice_kit_api.AdviceDefinition {
	return advice.GetAdviceDescriptionSingleAzureZone(SingleAzureZoneID, StatefulSetTargetType, "statefulset")
}




