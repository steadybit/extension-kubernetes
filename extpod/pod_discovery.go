// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extpod

import (
	"context"
	"fmt"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/steadybit/extension-kubernetes/v2/extnamespace"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"strings"
	"time"
)

type podDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.TargetDescriber = (*podDiscovery)(nil)
)

func NewPodDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	discovery := &podDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s, reflect.TypeOf(corev1.Pod{}))
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, time.Duration(extconfig.Config.DiscoveryRefreshThrottle)*time.Second),
	)
}

func (*podDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: PodTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (*podDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       PodTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes Pod", Other: "Kubernetes Pods"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cGF0aCBmaWxsLXJ1bGU9ImV2ZW5vZGQiIGNsaXAtcnVsZT0iZXZlbm9kZCIgZD0iTTEwLjQ0OCAyLjY1Nkw0LjY1NSA1LjU2Yy0uMDY2LjAzNC0uMTMxLjA3LS4xOTUuMTA3bDYuNTY5IDMuNjVhMiAyIDAgMDAxLjk0MiAwbDYuNTMtMy42MjhjLS4wNy0uMDQtLjE0LS4wNzgtLjIxNC0uMTEzTDEzLjA4IDIuNjI4YTMgMyAwIDAwLTIuNjMxLjAyOHptMTAuMzY2IDQuNTkxbC02Ljg3MSAzLjgxOGE0IDQgMCAwMS0uOTQzLjM3NnY5Ljk2N2wuMDgtLjAzNiA2LjIwNy0yLjk0OUEzIDMgMCAwMDIxIDE1LjcxM1Y4LjI4NmMwLS4zNi0uMDY1LS43MTItLjE4Ni0xLjAzOXpNMTEgMjEuNTU1VjExLjQ0MWE0IDQgMCAwMS0uOTQzLS4zNzZMMy4xNzIgNy4yMzlBMi45OTcgMi45OTcgMCAwMDMgOC4yNDJ2Ny41MTZhMyAzIDAgMDAxLjY1NSAyLjY4MWw1Ljc5MyAyLjkwNGMuMTc4LjA5LjM2My4xNi41NTIuMjEyeiIgZmlsbD0iY3VycmVudENvbG9yIi8+PC9zdmc+"),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.pod.name"},
				{Attribute: "k8s.cluster-name"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.deployment", FallbackAttributes: extutil.Ptr([]string{"k8s.statefulset", "k8s.daemonset"})},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.pod.name",
					Direction: "ASC",
				},
			},
		},
	}
}

func (p *podDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	pods := p.k8s.Pods()

	filteredPods := make([]*corev1.Pod, 0, len(pods))
	for _, pod := range pods {
		if client.IsExcludedFromDiscovery(pod.ObjectMeta) {
			continue
		}
		filteredPods = append(filteredPods, pod)
	}

	nodes := p.k8s.Nodes()
	targets := make([]discovery_kit_api.Target, len(filteredPods))
	for i, pod := range filteredPods {
		hostname, fqdn := extcommon.GetNodeHostnameAndFQDNs(nodes, pod.Spec.NodeName)
		attributes := map[string][]string{
			"k8s.pod.name":     {pod.Name},
			"k8s.namespace":    {pod.Namespace},
			"k8s.cluster-name": {extconfig.Config.ClusterName},
			"k8s.node.name":    {pod.Spec.NodeName},
			"host.hostname":    {hostname},
			"host.domainname":  fqdn,
		}

		for key, value := range pod.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}
		extnamespace.AddNamespaceLabels(p.k8s, pod.Namespace, attributes)
		extcommon.AddNodeLabels(p.k8s.Nodes(), pod.Spec.NodeName, attributes)

		var containerIds []string
		var containerIdsWithoutPrefix []string
		for _, container := range pod.Status.ContainerStatuses {
			if container.ContainerID == "" {
				continue
			}
			containerIds = append(containerIds, container.ContainerID)
			containerIdsWithoutPrefix = append(containerIdsWithoutPrefix, strings.SplitAfter(container.ContainerID, "://")[1])
		}
		if len(containerIds) > 0 {
			attributes["k8s.container.id"] = containerIds
		}
		if len(containerIdsWithoutPrefix) > 0 {
			attributes["k8s.container.id.stripped"] = containerIdsWithoutPrefix
		}

		ownerReferences := client.OwnerReferences(p.k8s, &pod.ObjectMeta)
		for _, ownerRef := range ownerReferences.OwnerRefs {
			attributes[fmt.Sprintf("k8s.%v", ownerRef.Kind)] = []string{ownerRef.Name}
			attributes["k8s.workload-type"] = []string{ownerRef.Kind}
			attributes["k8s.workload-owner"] = []string{ownerRef.Name}
		}

		services := p.k8s.ServicesMatchingToPodLabels(pod.Namespace, pod.ObjectMeta.Labels)
		if len(services) > 0 {
			var serviceNames = make([]string, 0, len(services))
			for _, service := range services {
				serviceNames = append(serviceNames, service.Name)
			}
			attributes["k8s.service.name"] = serviceNames
		}
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, pod.Namespace, pod.Name)
		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: PodTargetType,
			Label:      pod.Name,
			Attributes: attributes,
		}
	}
	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesPod), nil
}
