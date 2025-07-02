// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extstatefulset

import (
	"context"
	"fmt"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/steadybit/extension-kubernetes/v2/extnamespace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
	"reflect"
	"time"
)

type statefulSetDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.TargetDescriber          = (*statefulSetDiscovery)(nil)
	_ discovery_kit_sdk.EnrichmentRulesDescriber = (*statefulSetDiscovery)(nil)
)

func NewStatefulSetDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	discovery := &statefulSetDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s, reflect.TypeOf(corev1.Pod{}), reflect.TypeOf(appsv1.StatefulSet{}))
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, time.Duration(extconfig.Config.DiscoveryRefreshThrottle)*time.Second),
	)
}

func (d *statefulSetDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: StatefulSetTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *statefulSetDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       StatefulSetTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes StatefulSet", Other: "Kubernetes StatefulSets"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xOS41NjI1IDEzLjI1VjExLjc1SDIxQzIxLjI3NjEgMTEuNzUgMjEuNSAxMS41MjYxIDIxLjUgMTEuMjVWOS4zMTI1SDIzVjExLjI1QzIzIDEyLjM1NDYgMjIuMTA0NiAxMy4yNSAyMSAxMy4yNUgxOS41NjI1Wk0yMyA1LjQzNzVIMjEuNVYzLjVDMjEuNSAzLjIyMzg2IDIxLjI3NjEgMyAyMSAzSDE5LjU2MjVWMS41SDIxQzIyLjEwNDYgMS41IDIzIDIuMzk1NDMgMjMgMy41VjUuNDM3NVpNMTYuNjg3NSAxLjVWM0gxMy44MTI1VjEuNUgxNi42ODc1Wk0xMC45Mzc1IDEuNVYzSDkuNUM5LjIyMzg2IDMgOSAzLjIyMzg2IDkgMy41VjUuNDM3NUg3LjVWMy41QzcuNSAyLjM5NTQzIDguMzk1NDMgMS41IDkuNSAxLjVIMTAuOTM3NVpNMTcgMTEuMzYxMlY5LjkwMDE3QzE3IDguNjEyNjIgMTMuNjQwNiA3LjU2MjUgOS41IDcuNTYyNUM1LjM1OTM2IDcuNTYyNSAyIDguNjEyNjIgMiA5LjkwMDE3VjExLjM2MTJDMiAxMi42NDg3IDUuMzU5MzYgMTMuNjk4OSA5LjUgMTMuNjk4OUMxMy42NDA2IDEzLjY5ODkgMTcgMTIuNjQ4NyAxNyAxMS4zNjEyWk0xNyAxOC45NzQ4VjE0LjQzNzVDMTUuMzg4NiAxNS40OTY4IDEyLjQzOTUgMTUuOTg5OSA5LjUgMTUuOTg5OUM2LjU2MDU0IDE1Ljk4OTkgMy42MTEzMyAxNS40OTY4IDIgMTQuNDM3NVYxOC45NzQ4QzIgMjAuMjYyNCA1LjM1OTM2IDIxLjMxMjUgOS41IDIxLjMxMjVDMTMuNjQwNiAyMS4zMTI1IDE3IDIwLjI2MjQgMTcgMTguOTc0OFoiIGZpbGw9IiMxRDI2MzIiLz4KPC9zdmc+Cg=="),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.statefulset"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.statefulset",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *statefulSetDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	statefulsets := d.k8s.StatefulSets()

	filteredStatefulSets := make([]*appsv1.StatefulSet, 0, len(statefulsets))
	for _, sts := range statefulsets {
		if client.IsExcludedFromDiscovery(sts.ObjectMeta) {
			continue
		}
		filteredStatefulSets = append(filteredStatefulSets, sts)
	}

	nodes := d.k8s.Nodes()
	targets := make([]discovery_kit_api.Target, len(filteredStatefulSets))
	for i, sts := range filteredStatefulSets {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, sts.Namespace, sts.Name)
		attributes := map[string][]string{
			"k8s.namespace":      {sts.Namespace},
			"k8s.statefulset":    {sts.Name},
			"k8s.workload-type":  {"statefulset"},
			"k8s.workload-owner": {sts.Name},
			"k8s.cluster-name":   {extconfig.Config.ClusterName},
			"k8s.distribution":   {d.k8s.Distribution},
		}

		if sts.Spec.Replicas != nil {
			attributes["k8s.specification.replicas"] = []string{fmt.Sprintf("%d", *sts.Spec.Replicas)}
		}

		for key, value := range sts.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}
		extnamespace.AddNamespaceLabels(d.k8s, sts.Namespace, attributes)
		for key, value := range extcommon.GetPodBasedAttributes("statefulset", sts.ObjectMeta, d.k8s.PodsByLabelSelector(sts.Spec.Selector, sts.Namespace), nodes) {

			attributes[key] = value
		}
		for key, value := range extcommon.GetServiceNames(d.k8s.ServicesMatchingToPodLabels(sts.Namespace, sts.Spec.Template.Labels)) {
			attributes[key] = value
		}

		if !extconfig.Config.DisableAdvice {
			for key, value := range extcommon.GetKubeScoreForStatefulSet(sts, d.k8s.ServicesMatchingToPodLabels(sts.Namespace, sts.Spec.Template.Labels)) {
				attributes[key] = value
			}
		}

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: StatefulSetTargetType,
			Label:      sts.Name,
			Attributes: attributes,
		}
	}
	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesStatefulSet), nil
}

func (d *statefulSetDiscovery) DescribeEnrichmentRules() []discovery_kit_api.TargetEnrichmentRule {
	return []discovery_kit_api.TargetEnrichmentRule{
		getStatefulSetToContainerEnrichmentRule(),
	}
}

func getStatefulSetToContainerEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-statefulset-to-container",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: StatefulSetTargetType,
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
				Matcher: discovery_kit_api.Regex,
				Name:    "^k8s\\.label\\.(?!topology).*",
			},
		},
	}
}
