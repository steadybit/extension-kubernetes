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
		Icon:     extutil.Ptr(nodeIcon),
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
