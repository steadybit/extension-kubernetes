// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	discovery_kit_api "github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"net/http"
)

func RegisterContainerDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/container/discovery", exthttp.GetterAsHandler(getContainerDiscoveryDescription))
	exthttp.RegisterHttpHandler("/container/discovery/target-description", exthttp.GetterAsHandler(getContainerTargetDescription))
	exthttp.RegisterHttpHandler("/container/discovery/discovered-targets", getContainerDeployments)
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
					//TODO Service references
					//TODO Container Image overwrite
					//TODO POD-Labels
					//TODO Owner-References

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
					//TODO Labels
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

func getContainerDeployments(w http.ResponseWriter, r *http.Request, _ []byte) {
	targets := make([]discovery_kit_api.Target, 0)
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}
