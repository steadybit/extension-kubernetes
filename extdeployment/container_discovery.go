// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"fmt"
	discovery_kit_api "github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
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
		Id:         containerTargetType,
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
		Id:       containerTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes container", Other: "Kubernetes container"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  "1.0.0-SNAPSHOT",
		Icon:     extutil.Ptr(containerIcon),
		EnrichmentRules: extutil.Ptr([]discovery_kit_api.TargetEnrichmentRule{
			{
				Src: discovery_kit_api.SourceOrDestination{
					Type: containerTargetType,
					Selector: map[string]string{
						"container.id": "${dest.container.id}",
					},
				},
				Dest: discovery_kit_api.SourceOrDestination{
					Type: "container",
					Selector: map[string]string{
						"container.id": "${src.container.id}",
					},
				},
				Attributes: []discovery_kit_api.Attribute{
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.cluster-name",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.distribution",
					}, {
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.namespace",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.container.name",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.container.ready",
					},
					{
						//TODO --> Do we need an explicit "overwrite"?
						AggregationType: discovery_kit_api.Any,
						Name:            "container.image",
					},
					{
						AggregationType: discovery_kit_api.All,
						Name:            "k8s.service.name",
					},
					{
						AggregationType: discovery_kit_api.All,
						Name:            "k8s.service.namespace",
					},
					//TODO POD-Labels --> Do we need wildcards?
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.replicaset",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.daemonset",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.deployment",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.statefulset",
					},
				},
			}, {
				Src: discovery_kit_api.SourceOrDestination{
					Type: containerTargetType,
					Selector: map[string]string{
						"host.hostname": "${dest.host.hostname}",
					},
				},
				Dest: discovery_kit_api.SourceOrDestination{
					Type: "host",
					Selector: map[string]string{
						"host.hostname": "${src.host.hostname}",
					},
				},
				Attributes: []discovery_kit_api.Attribute{
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.cluster-name",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.distribution",
					},
					//TODO Labels  --> Do we need wildcards?
				},
			},
		}),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.container.name"},
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
	var targets []discovery_kit_api.Target

	for _, pod := range client.K8S.Pods() {
		podMetadata := pod.ObjectMeta
		ownerReferenceList := client.OwnerReferenceList(&podMetadata)
		fmt.Printf("Pod: %s \n ReferenceList:%+v\n", pod.Name, ownerReferenceList)

		for _, container := range pod.Status.ContainerStatuses {
			if container.ContainerID == "" {
				continue
			}

			containerIdWithoutPrefix := strings.SplitAfter(container.ContainerID, "://")[1]

			attributes := map[string][]string{
				"container.id":          {containerIdWithoutPrefix},
				"container.image":       {"TODO"},
				"k8s.cluster-name":      {"TODO"},
				"k8s.distribution":      {"TODO"},
				"k8s.namespace":         {podMetadata.Namespace},
				"k8s.container.name":    {container.Name},
				"k8s.container.ready":   {strconv.FormatBool(container.Ready)},
				"k8s.service.name":      {"TODO"},
				"k8s.service.namespace": {"TODO"},
			}

			for _, ownerRef := range ownerReferenceList {
				attributes[fmt.Sprintf("k8s.%v", ownerRef.Kind)] = []string{ownerRef.Name}
			}

			targets = append(targets, discovery_kit_api.Target{
				Id:         containerIdWithoutPrefix,
				Label:      container.Name,
				TargetType: containerTargetType,
				Attributes: attributes,
			})

		}

	}

	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}
