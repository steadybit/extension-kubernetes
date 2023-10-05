// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extstatefulset

import (
	"fmt"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/strings/slices"
	"net/http"
	"strings"
)

func RegisterStatefulSetDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/statefulset/discovery", exthttp.GetterAsHandler(getStatefulSetDiscoveryDescription))
	exthttp.RegisterHttpHandler("/statefulset/discovery/target-description", exthttp.GetterAsHandler(getStatefulSetTargetDescription))
	exthttp.RegisterHttpHandler("/statefulset/discovery/discovered-targets", getDiscoveredStatefulSets)
	exthttp.RegisterHttpHandler("/statefulset/discovery/rules/k8s-statefulset-to-container", exthttp.GetterAsHandler(getStatefulSetToContainerEnrichmentRule))
}

func getStatefulSetDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         StatefulSetTargetType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         "/statefulset/discovery/discovered-targets",
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func getStatefulSetTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       StatefulSetTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes StatefulSet", Other: "Kubernetes StatefulSets"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(statefulSetIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.statefulset"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.statefulset",
					Direction: "ASC",
				},
			},
		},
	}
}

func getDiscoveredStatefulSets(w http.ResponseWriter, _ *http.Request, _ []byte) {
	targets := getDiscoveredStatefulSetTargets(client.K8S)
	exthttp.WriteBody(w, discovery_kit_api.DiscoveryData{Targets: &targets})
}

func getDiscoveredStatefulSetTargets(k8s *client.Client) []discovery_kit_api.Target {
	statefulsets := k8s.StatefulSets()

	filteredStatefulSets := make([]*appsv1.StatefulSet, 0, len(statefulsets))
	if extconfig.Config.DisableDiscoveryExcludes {
		filteredStatefulSets = statefulsets
	} else {
		for _, sts := range statefulsets {
			if client.IsExcludedFromDiscovery(sts.ObjectMeta) {
				continue
			}
			filteredStatefulSets = append(filteredStatefulSets, sts)
		}
	}

	targets := make([]discovery_kit_api.Target, len(filteredStatefulSets))
	for i, sts := range filteredStatefulSets {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, sts.Namespace, sts.Name)
		attributes := map[string][]string{
			"k8s.namespace":    {sts.Namespace},
			"k8s.statefulset":  {sts.Name},
			"k8s.cluster-name": {extconfig.Config.ClusterName},
			"k8s.distribution": {k8s.Distribution},
		}

		for key, value := range sts.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		pods := k8s.PodsByLabelSelector(sts.Spec.Selector, sts.Namespace)
		if len(pods) > 0 {
			podNames := make([]string, len(pods))
			var containerIds []string
			var containerIdsWithoutPrefix []string
			var hostnames []string
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
		}

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: StatefulSetTargetType,
			Label:      sts.Name,
			Attributes: attributes,
		}
	}
	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesStatefulSet)
}

func getStatefulSetToContainerEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-statefulset-to-container",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: StatefulSetTargetType,
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
