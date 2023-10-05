// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extdeployment

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

func RegisterDeploymentDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/deployment/discovery", exthttp.GetterAsHandler(getDeploymentDiscoveryDescription))
	exthttp.RegisterHttpHandler("/deployment/discovery/target-description", exthttp.GetterAsHandler(getDeploymentTargetDescription))
	exthttp.RegisterHttpHandler("/deployment/discovery/discovered-targets", getDiscoveredDeployments)
	exthttp.RegisterHttpHandler("/deployment/discovery/rules/k8s-deployment-to-container", exthttp.GetterAsHandler(getDeploymentToContainerEnrichmentRule))
	exthttp.RegisterHttpHandler("/deployment/discovery/rules/container-to-k8s-deployment", exthttp.GetterAsHandler(getContainerToDeploymentEnrichmentRule))
}

func getDeploymentDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         DeploymentTargetType,
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
		Id:       DeploymentTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes Deployment", Other: "Kubernetes Deployments"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
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

func getDiscoveredDeployments(w http.ResponseWriter, _ *http.Request, _ []byte) {
	targets := getDiscoveredDeploymentTargets(client.K8S)
	exthttp.WriteBody(w, discovery_kit_api.DiscoveryData{Targets: &targets})
}

func getDiscoveredDeploymentTargets(k8s *client.Client) []discovery_kit_api.Target {
	deployments := k8s.Deployments()

	filteredDeployments := make([]*appsv1.Deployment, 0, len(deployments))
	if extconfig.Config.DisableDiscoveryExcludes {
		filteredDeployments = deployments
	} else {
		for _, d := range deployments {
			if client.IsExcludedFromDiscovery(d.ObjectMeta) {
				continue
			}
			filteredDeployments = append(filteredDeployments, d)
		}
	}

	targets := make([]discovery_kit_api.Target, len(filteredDeployments))
	for i, d := range filteredDeployments {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, d.Namespace, d.Name)
		attributes := map[string][]string{
			"k8s.namespace":    {d.Namespace},
			"k8s.deployment":   {d.Name},
			"k8s.cluster-name": {extconfig.Config.ClusterName},
			"k8s.distribution": {k8s.Distribution},
		}

		for key, value := range d.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.deployment.label.%v", key)] = []string{value}
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		pods := k8s.PodsByLabelSelector(d.Spec.Selector, d.Namespace)
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
			TargetType: DeploymentTargetType,
			Label:      d.Name,
			Attributes: attributes,
		}
	}
	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesDeployment)
}

func getDeploymentToContainerEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-deployment-to-container",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: DeploymentTargetType,
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
				Name:    "k8s.deployment.label.",
			},
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.label.",
			},
		},
	}
}

// Can be removed in the future (enrichment rules are currently not deleted automatically, therefore we need to "disable" the rule by making it non-matching)
func getContainerToDeploymentEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.container-to-kubernetes-deployment",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: "com.steadybit.ignore-me",
			Selector: map[string]string{
				"ignore": "${dest.ignore}",
			},
		},
		Dest: discovery_kit_api.SourceOrDestination{
			Type: "com.steadybit.ignore-me",
			Selector: map[string]string{
				"ignore": "${src.ignore}",
			},
		},
		Attributes: []discovery_kit_api.Attribute{},
	}
}
