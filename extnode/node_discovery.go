// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extnode

import (
	"fmt"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"strings"
)

func RegisterNodeDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/node/discovery", exthttp.GetterAsHandler(getNodeDiscoveryDescription))
	exthttp.RegisterHttpHandler("/node/discovery/target-description", exthttp.GetterAsHandler(getNodeTargetDescription))
	exthttp.RegisterHttpHandler("/node/discovery/discovered-targets", getDiscoveredNodes)
	exthttp.RegisterHttpHandler("/node/discovery/rules/k8s-node-to-host", exthttp.GetterAsHandler(getNodeToHostEnrichmentRule))
}

func getNodeDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         NodeTargetType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         "/node/discovery/discovered-targets",
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func getNodeTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       NodeTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes Node", Other: "Kubernetes Nodes"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr("data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.65%202.064a.993.993%200%2001.7%200l10%203.776a1.01%201.01%200%20010%201.889l-10%203.773a.993.993%200%2001-.7%200l-10-3.773A1.008%201.008%200%20011%206.784c0-.42.259-.796.65-.944l10-3.776zM1.063%2017.03a.998.998%200%20011.287-.591L12%2020.082l9.649-3.644a.998.998%200%20011.287.59%201.01%201.01%200%2001-.586%201.299l-10%203.776a.993.993%200%2001-.7%200l-10-3.776a1.01%201.01%200%2001-.586-1.298zm1.287-5.89a.998.998%200%2000-1.287.59%201.01%201.01%200%2000.586%201.299l10%203.776a.993.993%200%2000.7%200l10-3.776a1.01%201.01%200%2000.586-1.298.998.998%200%2000-1.287-.59L12%2014.782l-9.649-3.644z%22%20fill%3D%22currentColor%22%2F%3E%3C%2Fsvg%3E"),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.node.name"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.node.name",
					Direction: "ASC",
				},
			},
		},
	}
}

func getNodeToHostEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-node-to-host",
		Version: extbuild.GetSemverVersionStringOrUnknown(),

		Src: discovery_kit_api.SourceOrDestination{
			Type: NodeTargetType,
			Selector: map[string]string{
				"k8s.node.name": "${dest.host.hostname}",
			},
		},
		Dest: discovery_kit_api.SourceOrDestination{
			Type: "com.steadybit.extension_host.host",
			Selector: map[string]string{
				"host.hostname": "${src.k8s.node.name}",
			},
		},
		Attributes: []discovery_kit_api.Attribute{
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.cluster-name",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.distribution",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.namespace",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.replicaset",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.daemonset",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.deployment",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.statefulset",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.pod.name",
			},
		},
	}
}

func getDiscoveredNodes(w http.ResponseWriter, _ *http.Request, _ []byte) {
	targets := getDiscoveredNodeTargets(client.K8S)
	exthttp.WriteBody(w, discovery_kit_api.DiscoveryData{Targets: &targets})
}
func getDiscoveredNodeTargets(k8s *client.Client) []discovery_kit_api.Target {
	nodes := k8s.Nodes()

	filteredNodes := make([]*corev1.Node, 0, len(nodes))
	if extconfig.Config.DisableDiscoveryExcludes {
		filteredNodes = nodes
	} else {
		for _, d := range nodes {
			if client.IsExcludedFromDiscovery(d.ObjectMeta) {
				continue
			}
			filteredNodes = append(filteredNodes, d)
		}
	}

	targets := make([]discovery_kit_api.Target, len(filteredNodes))
	for i, node := range filteredNodes {
		attributes := map[string][]string{
			"k8s.node.name":    {node.Name},
			"k8s.cluster-name": {extconfig.Config.ClusterName},
			"host.hostname":    {node.Name},
			"k8s.distribution": {k8s.Distribution},
		}

		for key, value := range node.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		pods := k8s.Pods()
		if len(pods) > 0 {
			var podNames []string
			var containerIds []string
			var containerIdsWithoutPrefix []string
			deployments := make(map[string]bool)
			statefulSets := make(map[string]bool)
			daemonSets := make(map[string]bool)
			replicaSets := make(map[string]bool)
			namespaces := make(map[string]bool)
			for _, pod := range pods {
				if pod.Spec.NodeName == node.Name && !client.IsExcludedFromDiscovery(pod.ObjectMeta) {
					podNames = append(podNames, pod.Name)
					for _, container := range pod.Status.ContainerStatuses {
						if container.ContainerID == "" {
							continue
						}
						containerIds = append(containerIds, container.ContainerID)
						containerIdsWithoutPrefix = append(containerIdsWithoutPrefix, strings.SplitAfter(container.ContainerID, "://")[1])
					}
					namespaces[pod.Namespace] = true
					ownerReferences := client.OwnerReferences(k8s, &pod.ObjectMeta)
					for _, ownerReference := range ownerReferences.OwnerRefs {
						if ownerReference.Kind == "replicaset" {
							replicaSets[ownerReference.Name] = true
						}
						if ownerReference.Kind == "statefulset" {
							statefulSets[ownerReference.Name] = true
						}
						if ownerReference.Kind == "deployment" {
							deployments[ownerReference.Name] = true
						}
						if ownerReference.Kind == "daemonset" {
							daemonSets[ownerReference.Name] = true
						}
					}
				}
			}
			if len(containerIds) > 0 {
				attributes["k8s.container.id"] = containerIds
			}
			if len(containerIdsWithoutPrefix) > 0 {
				attributes["k8s.container.id.stripped"] = containerIdsWithoutPrefix
			}
			if len(podNames) > 0 {
				attributes["k8s.pod.name"] = podNames
			}
			if len(replicaSets) > 0 {
				attributes["k8s.replicaset"] = keys(replicaSets)
			}
			if len(statefulSets) > 0 {
				attributes["k8s.statefulset"] = keys(statefulSets)
			}
			if len(deployments) > 0 {
				attributes["k8s.deployment"] = keys(deployments)
			}
			if len(daemonSets) > 0 {
				attributes["k8s.daemonset"] = keys(daemonSets)
			}
			if len(namespaces) > 0 {
				attributes["k8s.namespace"] = keys(namespaces)
			}
		}

		targets[i] = discovery_kit_api.Target{
			Id:         node.Name,
			TargetType: NodeTargetType,
			Label:      node.Name,
			Attributes: attributes,
		}
	}
	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesNode)
}

func keys(m map[string]bool) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}