// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"fmt"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/utils"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
)

func RegisterDeploymentDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/deployment/discovery", exthttp.GetterAsHandler(getDeploymentDiscoveryDescription))
	exthttp.RegisterHttpHandler("/deployment/discovery/target-description", exthttp.GetterAsHandler(getDeploymentTargetDescription))
	exthttp.RegisterHttpHandler("/deployment/discovery/attribute-descriptions", exthttp.GetterAsHandler(getDeploymentAttributeDescriptions))
	exthttp.RegisterHttpHandler("/deployment/discovery/discovered-targets", getDiscoveredDeployments)
}

func getDeploymentDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         deploymentTargetId,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         "/deployment/discovery/discovered-targets",
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func getDeploymentTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       deploymentTargetId,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes deployment (ext)", Other: "Kubernetes deployments (ext)"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  "1.0.0-SNAPSHOT",
		Icon:     extutil.Ptr(deploymentIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.deployment"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.deployment",
					Direction: "ASC",
				},
			},
		},
	}
}

func getDeploymentAttributeDescriptions() discovery_kit_api.AttributeDescriptions {
	return discovery_kit_api.AttributeDescriptions{
		Attributes: []discovery_kit_api.AttributeDescription{
			{
				Attribute: "k8s.namespace",
				Label: discovery_kit_api.PluralLabel{
					One:   "namespace name",
					Other: "namespace names",
				},
			},
			{
				Attribute: "k8s.cluster-name",
				Label: discovery_kit_api.PluralLabel{
					One:   "cluster name",
					Other: "cluster names",
				},
			},
			{
				Attribute: "k8s.deployment",
				Label: discovery_kit_api.PluralLabel{
					One:   "deployment name",
					Other: "deployment names",
				},
			},
		},
	}
}

func getDiscoveredDeployments(w http.ResponseWriter, r *http.Request, _ []byte) {
	var deployments, err = utils.DeploymentLister.List(labels.Everything())
	if err != nil {
		panic(err.Error())
	}

	targets := make([]discovery_kit_api.Target, len(deployments))
	for i, d := range deployments {
		//TODO Implement Cluster-Name Agent Config
		targetName := fmt.Sprintf("%s/%s/%s", "test", d.Namespace, d.Name)

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: deploymentTargetId,
			Label:      d.Name,
			//TODO Add other attributes
			Attributes: map[string][]string{"k8s.namespace": {d.Name}},
		}
	}

	fmt.Printf("There are %d deployments in the cluster\n", len(deployments))
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}
