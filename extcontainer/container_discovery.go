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
		Icon:    extutil.Ptr(kubernetesContainerIcon),
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
