// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcontainer

import (
	"fmt"
	discovery_kit_api "github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"net/http"
	"strconv"
	"strings"
)

func RegisterContainerDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/container/discovery", exthttp.GetterAsHandler(getContainerDiscoveryDescription))
	exthttp.RegisterHttpHandler("/container/discovery/target-description", exthttp.GetterAsHandler(getContainerTargetDescription))
	exthttp.RegisterHttpHandler("/container/discovery/discovered-targets", getDiscoveredContainer)
}

func getContainerDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         KubernetesContainerTargetType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         "/container/discovery/discovered-targets",
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func getContainerTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       KubernetesContainerTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes Container", Other: "Kubernetes Containers"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(kubernetesContainerIcon),
		EnrichmentRules: extutil.Ptr([]discovery_kit_api.TargetEnrichmentRule{
			{
				Src: discovery_kit_api.SourceOrDestination{
					Type: KubernetesContainerTargetType,
					Selector: map[string]string{
						"k8s.container.id.stripped": "${dest.container.id.stripped}",
					},
				},
				Dest: discovery_kit_api.SourceOrDestination{
					Type: "container",
					Selector: map[string]string{
						"container.id.stripped": "${src.k8s.container.id.stripped}",
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
					}, {
						Matcher: discovery_kit_api.Equals,
						Name:    "k8s.namespace",
					},
					{
						Matcher: discovery_kit_api.Equals,
						Name:    "k8s.container.name",
					},
					{
						Matcher: discovery_kit_api.Equals,
						Name:    "k8s.container.ready",
					},
					{
						Matcher: discovery_kit_api.Equals,
						Name:    "k8s.container.image",
					},
					{
						Matcher: discovery_kit_api.Equals,
						Name:    "k8s.service.name",
					},
					{
						Matcher: discovery_kit_api.Equals,
						Name:    "k8s.service.namespace",
					},
					{
						Matcher: discovery_kit_api.StartsWith,
						Name:    "k8s.pod.label.",
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
				},
			}, {
				Src: discovery_kit_api.SourceOrDestination{
					Type: KubernetesContainerTargetType,
					Selector: map[string]string{
						"k8s.node.name": "${dest.host.hostname}",
					},
				},
				Dest: discovery_kit_api.SourceOrDestination{
					Type: "host",
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
			},
		}),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.container.name"},
				{Attribute: "k8s.pod.name"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.container.name",
					Direction: "ASC",
				},
			},
		},
	}
}

func getDiscoveredContainer(w http.ResponseWriter, r *http.Request, _ []byte) {
	targets := getDiscoveredContainerTargets(client.K8S)
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}

func getDiscoveredContainerTargets(k8s *client.Client) []discovery_kit_api.Target {
	pods := k8s.Pods()
	targets := make([]discovery_kit_api.Target, 0, len(pods))
	for _, pod := range pods {
		podMetadata := pod.ObjectMeta
		ownerReferences := client.OwnerReferences(k8s, &podMetadata)
		services := k8s.ServicesByPod(pod)

		for _, container := range pod.Status.ContainerStatuses {
			if container.ContainerID == "" {
				continue
			}

			containerIdWithoutPrefix := strings.SplitAfter(container.ContainerID, "://")[1]

			attributes := map[string][]string{
				"k8s.cluster-name":          {extconfig.Config.ClusterName},
				"k8s.container.id":          {container.ContainerID},
				"k8s.container.id.stripped": {containerIdWithoutPrefix},
				"k8s.container.name":        {container.Name},
				"k8s.container.ready":       {strconv.FormatBool(container.Ready)},
				"k8s.container.image":       {container.Image},
				"k8s.namespace":             {podMetadata.Namespace},
				"k8s.node.name":             {pod.Spec.NodeName},
				"k8s.pod.name":              {podMetadata.Name},
				"k8s.distribution":          {k8s.Distribution},
			}

			for key, value := range podMetadata.Labels {
				attributes[fmt.Sprintf("k8s.pod.label.%v", key)] = []string{value}
			}

			for _, service := range services {
				attributes["k8s.service.name"] = []string{service.Name}
				attributes["k8s.service.namespace"] = []string{service.Namespace}
			}

			for _, ownerRef := range ownerReferences.OwnerRefs {
				attributes[fmt.Sprintf("k8s.%v", ownerRef.Kind)] = []string{ownerRef.Name}
			}

			targets = append(targets, discovery_kit_api.Target{
				Id:         container.ContainerID,
				Label:      container.Name,
				TargetType: KubernetesContainerTargetType,
				Attributes: attributes,
			})
		}
	}
	return targets
}
