// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extnamespace

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
	"github.com/steadybit/extension-kubernetes/extdaemonset"
	"github.com/steadybit/extension-kubernetes/extdeployment"
	"github.com/steadybit/extension-kubernetes/extnode"
	"github.com/steadybit/extension-kubernetes/extpod"
	"github.com/steadybit/extension-kubernetes/extstatefulset"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"time"
)

type namespaceDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.EnrichmentRulesDescriber = (*namespaceDiscovery)(nil)
)

func NewNamespaceDiscovery(ctx context.Context, k8s *client.Client) discovery_kit_sdk.EnrichmentDataDiscovery {
	discovery := &namespaceDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s, reflect.TypeOf(corev1.Pod{}), reflect.TypeOf(corev1.Node{}))
	return discovery_kit_sdk.NewCachedEnrichmentDataDiscovery(
		discovery,
		discovery_kit_sdk.WithRefreshEnrichmentDataNow(),
		discovery_kit_sdk.WithRefreshEnrichmentDataTrigger(ctx, chRefresh, 5*time.Second),
	)
}

func (c *namespaceDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: KubernetesNamespaceEnrichmentDataType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (c *namespaceDiscovery) DescribeEnrichmentRules() []discovery_kit_api.TargetEnrichmentRule {
	return []discovery_kit_api.TargetEnrichmentRule{
		getNamespaceToXEnrichmentRule("com.steadybit.extension_container.container"),
		getNamespaceToXEnrichmentRule("com.steadybit.extension_jvm.jvm-instance"),
		getNamespaceToXEnrichmentRule("com.steadybit.extension_host.host"),
		getNamespaceToXEnrichmentRule(extdeployment.DeploymentTargetType),
		getNamespaceToXEnrichmentRule(extstatefulset.StatefulSetTargetType),
		getNamespaceToXEnrichmentRule(extdaemonset.DaemonSetTargetType),
		getNamespaceToXEnrichmentRule(extpod.PodTargetType),
		getNamespaceToXEnrichmentRule(extnode.NodeTargetType),
	}
}

func getNamespaceToXEnrichmentRule(destTargetType string) discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-namespace-to-"+destTargetType,
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: KubernetesNamespaceEnrichmentDataType,
			Selector: map[string]string{
				"k8s.namespace": "${dest.k8s.namespace}",
			},
		},
		Dest: discovery_kit_api.SourceOrDestination{
			Type: destTargetType,
			Selector: map[string]string{
				"k8s.namespace": "${src.k8s.namespace}",
			},
		},
		Attributes: []discovery_kit_api.Attribute{
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.namespace.label.",
			},
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.label.",
			},
		},
	}
}

func (c *namespaceDiscovery) DiscoverEnrichmentData(_ context.Context) ([]discovery_kit_api.EnrichmentData, error) {
	namespaces := c.k8s.Namespaces()

	filteredNamespaces := make([]*corev1.Namespace, 0, len(namespaces))
	if extconfig.Config.DisableDiscoveryExcludes {
		filteredNamespaces = namespaces
	} else {
		for _, p := range namespaces {
			if client.IsExcludedFromDiscovery(p.ObjectMeta) {
				continue
			}
			filteredNamespaces = append(filteredNamespaces, p)
		}
	}

	enrichmentDataList := make([]discovery_kit_api.EnrichmentData, 0, len(filteredNamespaces))
	for _, namespace := range filteredNamespaces {
		namespaceMetadata := namespace.ObjectMeta

		attributes := map[string][]string{
			"k8s.cluster-name": {extconfig.Config.ClusterName},
			"k8s.namespace.id": {string(namespace.UID)},
			"k8s.namespace":    {namespace.Name},
			"k8s.distribution": {c.k8s.Distribution},
		}

		for key, value := range namespaceMetadata.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.namespace.label.%v", key)] = []string{value}
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		enrichmentDataList = append(enrichmentDataList, discovery_kit_api.EnrichmentData{
			Id:                 string(namespace.UID),
			EnrichmentDataType: KubernetesNamespaceEnrichmentDataType,
			Attributes:         attributes,
		})
	}
	return discovery_kit_commons.ApplyAttributeExcludesToEnrichmentData(enrichmentDataList, extconfig.Config.DiscoveryAttributesExcludesNamespace), nil
}
