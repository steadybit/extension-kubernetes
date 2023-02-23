// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"fmt"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"net/http"
)

func RegisterDeploymentDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/deployment/discovery", exthttp.GetterAsHandler(getDeploymentDiscoveryDescription))
	exthttp.RegisterHttpHandler("/deployment/discovery/target-description", exthttp.GetterAsHandler(getDeploymentTargetDescription))
	exthttp.RegisterHttpHandler("/deployment/discovery/discovered-targets", getDiscoveredDeployments)
}

func getDeploymentDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         deploymentTargetType,
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
		Id:       deploymentTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes deployment", Other: "Kubernetes deployments"},
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

func getDiscoveredDeployments(w http.ResponseWriter, r *http.Request, _ []byte) {
	deployments := client.K8S.Deployments()

	targets := make([]discovery_kit_api.Target, len(deployments))
	for i, d := range deployments {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, d.Namespace, d.Name)
		attributes := map[string][]string{
			"k8s.namespace":    {d.Namespace},
			"k8s.deployment":   {d.Name},
			"k8s.cluster-name": {extconfig.Config.ClusterName},
			//"k8s.distribution":        {"TODO implement me"},
		}

		for key, value := range d.ObjectMeta.Labels {
			attributes[fmt.Sprintf("k8s.deployment.label.%v", key)] = []string{value}
		}

		pods := client.K8S.PodsByDeployment(d)
		if len(pods) > 0 {
			podNames := make([]string, len(pods))
			var containerIds []string
			for podIndex, pod := range pods {
				podNames[podIndex] = pod.Name
				for _, container := range pod.Status.ContainerStatuses {
					containerIds = append(containerIds, container.ContainerID)
				}
			}
			attributes["k8s.pod.name"] = podNames
			attributes["k8s.container.id"] = containerIds
		}

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: deploymentTargetType,
			Label:      d.Name,
			Attributes: attributes,
		}
	}
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}
