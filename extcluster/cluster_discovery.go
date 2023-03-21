// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcluster

import (
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"net/http"
)

func RegisterClusterDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/cluster/discovery", exthttp.GetterAsHandler(getClusterDiscoveryDescription))
	exthttp.RegisterHttpHandler("/cluster/discovery/target-description", exthttp.GetterAsHandler(getClusterTargetDescription))
	exthttp.RegisterHttpHandler("/cluster/discovery/discovered-targets", getDiscoveredCluster)
}

func getClusterDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         ClusterTargetType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         "/cluster/discovery/discovered-targets",
			CallInterval: extutil.Ptr("60m"),
		},
	}
}

func getClusterTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       ClusterTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes Cluster", Other: "Kubernetes Cluster"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(clusterIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.cluster-name",
					Direction: "ASC",
				},
			},
		},
	}
}

func getDiscoveredCluster(w http.ResponseWriter, r *http.Request, _ []byte) {
	targets := getDiscoveredClusterTargets()
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}

func getDiscoveredClusterTargets() []discovery_kit_api.Target {
	return []discovery_kit_api.Target{
		{
			Id:         extconfig.Config.ClusterName,
			Label:      extconfig.Config.ClusterName,
			TargetType: ClusterTargetType,
			Attributes: map[string][]string{
				"k8s.cluster-name": {extconfig.Config.ClusterName},
			},
		},
	}
}
