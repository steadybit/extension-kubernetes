// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcontainer

import (
	"context"
	"fmt"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcommon"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"strings"
	"time"
)

type containerDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.EnrichmentRulesDescriber = (*containerDiscovery)(nil)
)

func NewContainerDiscovery(ctx context.Context, k8s *client.Client) discovery_kit_sdk.EnrichmentDataDiscovery {
	discovery := &containerDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s, reflect.TypeOf(corev1.Pod{}), reflect.TypeOf(corev1.Node{}))
	return discovery_kit_sdk.NewCachedEnrichmentDataDiscovery(
		discovery,
		discovery_kit_sdk.WithRefreshEnrichmentDataNow(),
		discovery_kit_sdk.WithRefreshEnrichmentDataTrigger(ctx, chRefresh, 5*time.Second),
	)
}

func (c *containerDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         KubernetesContainerEnrichmentDataType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (c *containerDiscovery) DescribeEnrichmentRules() []discovery_kit_api.TargetEnrichmentRule {
	return []discovery_kit_api.TargetEnrichmentRule{
		getContainerToContainerEnrichmentRule(),
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
				Name:    "k8s.workload-type",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.workload-owner",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.statefulset",
			},
		},
	}
}

func (c *containerDiscovery) DiscoverEnrichmentData(_ context.Context) ([]discovery_kit_api.EnrichmentData, error) {
	pods := c.k8s.Pods()

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
		ownerReferences := client.OwnerReferences(c.k8s, &podMetadata)
		services := c.k8s.ServicesMatchingToPodLabels(pod.Namespace, pod.ObjectMeta.Labels)

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
				"k8s.distribution":          {c.k8s.Distribution},
			}

			for _, containerSpec := range pod.Spec.Containers {
				//TODO Remove these attributes when old weak spot feature is removed
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
	return discovery_kit_commons.ApplyAttributeExcludesToEnrichmentData(enrichmentDataList, extconfig.Config.DiscoveryAttributesExcludesContainer), nil
}
