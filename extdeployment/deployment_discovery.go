// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extdeployment

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcommon"
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
		Icon:     extutil.Ptr("data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A"),
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
			"k8s.namespace":                    {d.Namespace},
			"k8s.deployment":                   {d.Name},
			"k8s.cluster-name":                 {extconfig.Config.ClusterName},
			"k8s.distribution":                 {k8s.Distribution},
			"k8s.deployment.strategy":          {string(d.Spec.Strategy.Type)},
			"k8s.deployment.min-ready-seconds": {fmt.Sprintf("%d", d.Spec.MinReadySeconds)},
		}
		if k8s.Permissions().CanReadHorizontalPodAutoscalers() {
			hpa := k8s.HorizontalPodAutoscalerByNamespaceAndDeployment(d.Namespace, d.Name)
			attributes["k8s.deployment.hpa.existent"] = []string{fmt.Sprintf("%v", hpa != nil)}
		}

		if d.Spec.Replicas != nil {
			attributes["k8s.deployment.replicas"] = []string{fmt.Sprintf("%d", *d.Spec.Replicas)}
		}

		for key, value := range d.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.deployment.label.%v", key)] = []string{value}
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		pods := k8s.PodsByLabelSelector(d.Spec.Selector, d.Namespace)
		if len(pods) > extconfig.Config.DiscoveryMaxPodCount {
			log.Warn().Msgf("Deployment %s/%s has more than %d pods. Skip listing pods, containers and hosts.", d.Namespace, d.Name, extconfig.Config.DiscoveryMaxPodCount)
			attributes["k8s.pod.name"] = []string{"too-many-pods"}
			attributes["k8s.container.id"] = []string{"too-many-pods"}
			attributes["k8s.container.id.stripped"] = []string{"too-many-pods"}
			attributes["host.hostname"] = []string{"too-many-pods"}
		} else if len(pods) > 0 {
			podNames := make([]string, len(pods))
			var containerIds []string
			var containerIdsWithoutPrefix []string
			var containerNamesWithoutLimitCPU []string
			var containerNamesWithoutLimitMemory []string
			var containerWithoutLivenessProbe []string
			var containerWithoutReadinessProbe []string
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
				for _, containerSpec := range pod.Spec.Containers {
					if containerSpec.Resources.Limits.Cpu().MilliValue() == 0 {
						containerNamesWithoutLimitCPU = append(containerNamesWithoutLimitCPU, containerSpec.Name)
					}
					if containerSpec.Resources.Limits.Memory().MilliValue() == 0 {
						containerNamesWithoutLimitMemory = append(containerNamesWithoutLimitMemory, containerSpec.Name)
					}
					if containerSpec.LivenessProbe == nil {
						containerWithoutLivenessProbe = append(containerWithoutLivenessProbe, containerSpec.Name)
					}
					if containerSpec.ReadinessProbe == nil {
						containerWithoutReadinessProbe = append(containerWithoutReadinessProbe, containerSpec.Name)
					}
				}
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
			if len(containerNamesWithoutLimitCPU) > 0 {
				attributes["k8s.container.spec.name.limit.cpu.not-set"] = containerNamesWithoutLimitCPU
			}
			if len(containerNamesWithoutLimitMemory) > 0 {
				attributes["k8s.container.spec.name.limit.memory.not-set"] = containerNamesWithoutLimitMemory
			}
			if len(containerWithoutLivenessProbe) > 0 {
				attributes["k8s.container.probes.liveness.not-set"] = containerWithoutLivenessProbe
			}
			if len(containerWithoutReadinessProbe) > 0 {
				attributes["k8s.container.probes.readiness.not-set"] = containerWithoutReadinessProbe
			}
			scoreAttributes := extcommon.AddKubeScoreAttributesDeployment(d)
			for key, value := range scoreAttributes {
				attributes[key] = value
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
		Attributes: []discovery_kit_api.Attribute{
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "ignore.me",
			},
		},
	}
}
