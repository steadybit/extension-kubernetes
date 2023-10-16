// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcontainer

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

func RegisterContainerDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/container/discovery", exthttp.GetterAsHandler(getContainerDiscoveryDescription))
	exthttp.RegisterHttpHandler("/container/discovery/target-description", exthttp.GetterAsHandler(getContainerTargetDescription))
	exthttp.RegisterHttpHandler("/container/discovery/rules/k8s-container-to-container", exthttp.GetterAsHandler(getContainerToContainerEnrichmentRule))
	exthttp.RegisterHttpHandler("/container/discovery/rules/k8s-container-to-host", exthttp.GetterAsHandler(getContainerToHostEnrichmentRule))
	exthttp.RegisterHttpHandler("/container/discovery/discovered-enrichment-data", getDiscoveredContainer)
}

func getContainerDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         KubernetesContainerEnrichmentDataType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         "/container/discovery/discovered-enrichment-data",
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func getContainerTargetDescription() discovery_kit_api.TargetDescription {
	//We need to keep this for a while to make sure that enrichment rules from a previous version of this target description are deleted in the platform.
	return discovery_kit_api.TargetDescription{
		Id:      KubernetesContainerEnrichmentDataType,
		Label:   discovery_kit_api.PluralLabel{One: "Kubernetes Container (deprecated)", Other: "Kubernetes Containers (deprecated)"},
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Icon:    extutil.Ptr("data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A"),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.container.name"},
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

func getContainerToContainerEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-container-to-container",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: KubernetesContainerEnrichmentDataType,
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
				Name:    "k8s.container.id",
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
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.pod.label.",
			},
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.label.",
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
	}
}

// Can be removed in the future (enrichment rules are currently not deleted automatically, therefore we need to "disable" the rule by making it non-matching)
func getContainerToHostEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-container-to-host",
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
		}}
}

func getDiscoveredContainer(w http.ResponseWriter, _ *http.Request, _ []byte) {
	enrichmentData := getDiscoveredContainerEnrichmentData(client.K8S)
	exthttp.WriteBody(w, discovery_kit_api.DiscoveryData{EnrichmentData: &enrichmentData})
}

func getDiscoveredContainerEnrichmentData(k8s *client.Client) []discovery_kit_api.EnrichmentData {
	pods := k8s.Pods()

	filteredPods := make([]*corev1.Pod, 0, len(pods))
	if extconfig.Config.DisableDiscoveryExcludes {
		filteredPods = pods
	} else {
		for _, p := range pods {
			if client.IsExcludedFromDiscovery(p.ObjectMeta) {
				continue
			}
			filteredPods = append(filteredPods, p)
		}
	}

	enrichmentDataList := make([]discovery_kit_api.EnrichmentData, 0, len(filteredPods))
	for _, pod := range filteredPods {
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
				"k8s.container.image":       {container.Image},
				"k8s.namespace":             {podMetadata.Namespace},
				"k8s.node.name":             {pod.Spec.NodeName},
				"k8s.pod.name":              {podMetadata.Name},
				"k8s.distribution":          {k8s.Distribution},
			}

			for _, containerSpec := range pod.Spec.Containers {
				if containerSpec.Name == container.Name {
					attributes["k8s.container.limit.cpu"] = []string{fmt.Sprintf("%d", containerSpec.Resources.Limits.Cpu().MilliValue())}
					attributes["k8s.container.limit.memory"] = []string{fmt.Sprintf("%d", containerSpec.Resources.Limits.Memory().MilliValue())}
					attributes["k8s.container.image.pull-policy"] = []string{string(containerSpec.ImagePullPolicy)}
					attributes["k8s.container.probes.liveness.existent"] = []string{fmt.Sprintf("%t", containerSpec.LivenessProbe != nil)}
					attributes["k8s.container.probes.readiness.existent"] = []string{fmt.Sprintf("%t", containerSpec.ReadinessProbe != nil)}
					break
				}
			}
			for key, value := range podMetadata.Labels {
				if !slices.Contains(extconfig.Config.LabelFilter, key) {
					attributes[fmt.Sprintf("k8s.pod.label.%v", key)] = []string{value}
					attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
				}
			}

			if len(services) > 0 {
				var serviceNames = make([]string, 0, len(services))
				for _, service := range services {
					serviceNames = append(serviceNames, service.Name)
				}
				slices.Sort(serviceNames)
				attributes["k8s.service.name"] = serviceNames
			}

			for _, ownerRef := range ownerReferences.OwnerRefs {
				attributes[fmt.Sprintf("k8s.%v", ownerRef.Kind)] = []string{ownerRef.Name}
			}

			enrichmentDataList = append(enrichmentDataList, discovery_kit_api.EnrichmentData{
				Id:                 container.ContainerID,
				EnrichmentDataType: KubernetesContainerEnrichmentDataType,
				Attributes:         attributes,
			})
		}
	}
	return discovery_kit_commons.ApplyAttributeExcludesToEnrichmentData(enrichmentDataList, extconfig.Config.DiscoveryAttributesExcludesContainer)
}
