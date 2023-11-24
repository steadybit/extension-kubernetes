// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extdaemonset

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcommon"
	"github.com/steadybit/extension-kubernetes/extconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
	"reflect"
	"strings"
	"time"
)

type daemonSetDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.TargetDescriber          = (*daemonSetDiscovery)(nil)
	_ discovery_kit_sdk.EnrichmentRulesDescriber = (*daemonSetDiscovery)(nil)
)

func NewDaemonSetDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	discovery := &daemonSetDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s, reflect.TypeOf(corev1.Pod{}), reflect.TypeOf(appsv1.DaemonSet{}))

	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, 5*time.Second),
	)
}

func (d *daemonSetDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         DaemonSetTargetType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *daemonSetDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       DaemonSetTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes DaemonSet", Other: "Kubernetes DaemonSets"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xOS41NjI1IDExLjc1VjEzLjI1SDIxQzIyLjEwNDYgMTMuMjUgMjMgMTIuMzU0NiAyMyAxMS4yNVY5LjMxMjVIMjEuNVYxMS4yNUMyMS41IDExLjUyNjEgMjEuMjc2MSAxMS43NSAyMSAxMS43NUgxOS41NjI1Wk0yMS41IDUuNDM3NUgyM1YzLjVDMjMgMi4zOTU0MyAyMi4xMDQ2IDEuNSAyMSAxLjVIMTkuNTYyNVYzSDIxQzIxLjI3NjEgMyAyMS41IDMuMjIzODYgMjEuNSAzLjVWNS40Mzc1Wk0xNi42ODc1IDNWMS41SDEzLjgxMjVWM0gxNi42ODc1Wk0xMC45Mzc1IDNWMS41SDkuNUM4LjM5NTQzIDEuNSA3LjUgMi4zOTU0MyA3LjUgMy41VjUuNDM3NUg5VjMuNUM5IDMuMjIzODYgOS4yMjM4NiAzIDkuNSAzSDEwLjkzNzVaTTIgOC42ODc1QzIgOC40MTEzNiAyLjIyMzg2IDguMTg3NSAyLjUgOC4xODc1SDE2LjVDMTYuNzc2MSA4LjE4NzUgMTcgOC40MTEzNiAxNyA4LjY4NzVWMTYuNDM3NUMxNyAxNi43MTM2IDE2Ljc3NjEgMTYuOTM3NSAxNi41IDE2LjkzNzVIMi41QzIuMjIzODYgMTYuOTM3NSAyIDE2LjcxMzYgMiAxNi40Mzc1VjguNjg3NVpNMiAxOS4zMTI1QzIgMTkuMDM2NCAyLjIyMzg2IDE4LjgxMjUgMi41IDE4LjgxMjVIMTYuNUMxNi43NzYxIDE4LjgxMjUgMTcgMTkuMDM2NCAxNyAxOS4zMTI1VjIwLjE4NzVDMTcgMjAuNDYzNiAxNi43NzYxIDIwLjY4NzUgMTYuNSAyMC42ODc1SDIuNUMyLjIyMzg2IDIwLjY4NzUgMiAyMC40NjM2IDIgMjAuMTg3NVYxOS4zMTI1WiIgZmlsbD0iIzFEMjYzMiIvPgo8L3N2Zz4K"),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.daemonset"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.daemonset",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *daemonSetDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	daemonsets := d.k8s.DaemonSets()

	filteredDaemonSets := make([]*appsv1.DaemonSet, 0, len(daemonsets))
	if extconfig.Config.DisableDiscoveryExcludes {
		filteredDaemonSets = daemonsets
	} else {
		for _, ds := range daemonsets {
			if client.IsExcludedFromDiscovery(ds.ObjectMeta) {
				continue
			}
			filteredDaemonSets = append(filteredDaemonSets, ds)
		}
	}

	targets := make([]discovery_kit_api.Target, len(filteredDaemonSets))
	for i, ds := range filteredDaemonSets {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, ds.Namespace, ds.Name)
		attributes := map[string][]string{
			"k8s.namespace":    {ds.Namespace},
			"k8s.daemonset":    {ds.Name},
			"k8s.cluster-name": {extconfig.Config.ClusterName},
			"k8s.distribution": {d.k8s.Distribution},
		}

		for key, value := range ds.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		pods := d.k8s.PodsByLabelSelector(ds.Spec.Selector, ds.Namespace)
		if len(pods) > extconfig.Config.DiscoveryMaxPodCount {
			log.Warn().Msgf("Daemonset %s/%s has more than %d pods. Skip listing pods, containers and hosts.", ds.Namespace, ds.Name, extconfig.Config.DiscoveryMaxPodCount)
			attributes["k8s.pod.name"] = []string{"too-many-pods"}
			attributes["k8s.container.id"] = []string{"too-many-pods"}
			attributes["k8s.container.id.stripped"] = []string{"too-many-pods"}
			attributes["host.hostname"] = []string{"too-many-pods"}
		} else if len(pods) > 0 {
			podNames := make([]string, len(pods))
			var containerIds []string
			var containerIdsWithoutPrefix []string
			var hostnames []string
			var containerNamesWithoutLimitCPU []string
			var containerNamesWithoutLimitMemory []string
			var containerWithoutLivenessProbe []string
			var containerWithoutReadinessProbe []string
			var containerWithLatestTag []string
			var containerWithoutImagePullPolicyAlways []string
			for podIndex, pod := range pods {
				podNames[podIndex] = pod.Name
				for _, container := range pod.Status.ContainerStatuses {
					if container.ContainerID == "" {
						continue
					}
					containerIds = append(containerIds, container.ContainerID)
					containerIdsWithoutPrefix = append(containerIdsWithoutPrefix, strings.SplitAfter(container.ContainerID, "://")[1])
				}
				hostnames = append(hostnames, pod.Spec.NodeName)
				for _, containerSpec := range pod.Spec.Containers {
					if containerSpec.Resources.Limits.Cpu().MilliValue() == 0 {
						containerNamesWithoutLimitCPU = append(containerNamesWithoutLimitCPU, containerSpec.Name)
					}
					if containerSpec.Resources.Limits.Memory().MilliValue() == 0 {
						containerNamesWithoutLimitMemory = append(containerNamesWithoutLimitMemory, containerSpec.Name)
					}
					if containerSpec.LivenessProbe == nil {
						containerWithoutLivenessProbe = append(containerWithoutLivenessProbe, containerSpec.Name)
					}
					if containerSpec.ReadinessProbe == nil {
						containerWithoutReadinessProbe = append(containerWithoutReadinessProbe, containerSpec.Name)
					}
					if strings.HasSuffix(containerSpec.Image, "latest") {
						containerWithLatestTag = append(containerWithLatestTag, containerSpec.Image)
					}
					if containerSpec.ImagePullPolicy != "Always" {
						containerWithoutImagePullPolicyAlways = append(containerWithoutImagePullPolicyAlways, containerSpec.Image)
					}
				}
			}
			attributes["k8s.pod.name"] = podNames
			if len(containerIds) > 0 {
				attributes["k8s.container.id"] = containerIds
			}
			if len(containerIdsWithoutPrefix) > 0 {
				attributes["k8s.container.id.stripped"] = containerIdsWithoutPrefix
			}
			if len(hostnames) > 0 {
				attributes["host.hostname"] = hostnames
			}
			if len(containerNamesWithoutLimitCPU) > 0 {
				attributes["k8s.container.spec.name.limit.cpu.not-set"] = containerNamesWithoutLimitCPU
			}
			if len(containerNamesWithoutLimitMemory) > 0 {
				attributes["k8s.container.spec.name.limit.memory.not-set"] = containerNamesWithoutLimitMemory
			}
			if len(containerWithoutLivenessProbe) > 0 {
				attributes["k8s.container.probes.liveness.not-set"] = containerWithoutLivenessProbe
			}
			if len(containerWithoutReadinessProbe) > 0 {
				attributes["k8s.container.probes.readiness.not-set"] = containerWithoutReadinessProbe
			}
			if len(containerWithLatestTag) > 0 {
				attributes["k8s.container.image.with-latest-tag"] = containerWithLatestTag
			}
			if len(containerWithoutImagePullPolicyAlways) > 0 {
				attributes["k8s.container.image.without-image-pull-policy-always"] = containerWithoutImagePullPolicyAlways
			}

			scoreAttributes := addKubeScoreAttributesDaemonSet(ds)
			for key, value := range scoreAttributes {
				attributes[key] = value
			}
		}

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: DaemonSetTargetType,
			Label:      ds.Name,
			Attributes: attributes,
		}
	}
	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesDaemonSet), nil
}

func (d *daemonSetDiscovery) DescribeEnrichmentRules() []discovery_kit_api.TargetEnrichmentRule {
	return []discovery_kit_api.TargetEnrichmentRule{
		getDaemonSetToContainerEnrichmentRule(),
	}
}

func getDaemonSetToContainerEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-daemonset-to-container",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: DaemonSetTargetType,
			Selector: map[string]string{
				"k8s.container.id.stripped": "${dest.container.id.stripped}",
			},
		},
		Dest: discovery_kit_api.SourceOrDestination{
			Type: "com.steadybit.extension_container.container",
			Selector: map[string]string{
				"container.id.stripped": "${src.k8s.container.id.stripped}",
			},
		},
		Attributes: []discovery_kit_api.Attribute{
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.label.",
			},
		},
	}
}

func addKubeScoreAttributesDaemonSet(daemonSet *appsv1.DaemonSet) map[string][]string {
	apiVersion := "apps/v1"
	kind := "Deployment"
	if daemonSet.APIVersion != "" {
		apiVersion = daemonSet.APIVersion
	}
	if daemonSet.Kind != "" {
		kind = daemonSet.Kind
	}
	return extcommon.AddKubeScoreAttributes(daemonSet, daemonSet.Namespace, daemonSet.Name, apiVersion, kind)
}
