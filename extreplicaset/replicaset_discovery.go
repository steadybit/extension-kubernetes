// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extreplicaset

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type replicasetDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.TargetDescriber = (*replicasetDiscovery)(nil)
)

func NewReplicaSetDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	discovery := &replicasetDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s,
		reflect.TypeOf(corev1.Pod{}),
		reflect.TypeOf(appsv1.ReplicaSet{}),
		reflect.TypeOf(corev1.Service{}),
	)
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, time.Duration(extconfig.Config.DiscoveryRefreshThrottle)*time.Second),
	)
}

func (d *replicasetDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: ReplicaSetTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *replicasetDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       ReplicaSetTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes ReplicaSet", Other: "Kubernetes ReplicaSets"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEzLjIxMjkgMTEuNzEzOUMxMy42NDc0IDExLjcxMzkgMTQgMTIuMDY2NSAxNCAxMi41MDFWMTguOTI2OEMxNCAxOS4zNjEyIDEzLjY0NzQgMTkuNzEzOSAxMy4yMTI5IDE5LjcxMzlIMi43ODcxMUMyLjM1MjY1IDE5LjcxMzkgMiAxOS4zNjEyIDIgMTguOTI2OFYxMi41MDFDMiAxMi4wNjY1IDIuMzUyNjUgMTEuNzEzOSAyLjc4NzExIDExLjcxMzlIMTMuMjEyOVpNMTcuNTg3OSAxNS4xOTYzQzE3LjU4NzcgMTYuMDczNSAxNi44OTQgMTYuNzg0MiAxNi4wMzkxIDE2Ljc4NDJIMTQuOTI2OFYxNS41OTI4SDE2LjAzOTFDMTYuMjUyNyAxNS41OTI4IDE2LjQyNjUgMTUuNDE1NCAxNi40MjY4IDE1LjE5NjNWMTMuNjU3MkgxNy41ODc5VjE1LjE5NjNaTTIxLjk5OSA5LjY5OTIyVjExLjExMjNDMjEuOTk4OSAxMS45MTc5IDIxLjI1NjggMTIuNTcxMiAyMC4zNDA4IDEyLjU3MTNIMTkuMTQ4NFYxMS40NzY2SDIwLjM0MDhDMjAuNTY5NyAxMS40NzY0IDIwLjc1NDggMTEuMzEzNiAyMC43NTQ5IDExLjExMjNWOS42OTkyMkgyMS45OTlaTTguMjQ5MDIgOC42NDI1OEg3LjEzNjcyQzYuOTIyOTcgOC42NDI1OCA2Ljc0OTA3IDguODIwNzUgNi43NDkwMiA5LjA0MDA0VjEwLjU3OTFINS41ODc4OVY5LjA0MDA0QzUuNTg3OTQgOC4xNjI3MyA2LjI4MTY0IDcuNDUxMTcgNy4xMzY3MiA3LjQ1MTE3SDguMjQ5MDJWOC42NDI1OFpNMTYuMDM5MSA3LjQ1MTE3QzE2Ljg5NDEgNy40NTExNyAxNy41ODc4IDguMTYyNzMgMTcuNTg3OSA5LjA0MDA0VjEwLjU3OTFIMTYuNDI2OFY5LjA0MDA0QzE2LjQyNjcgOC44MjA3NSAxNi4yNTI4IDguNjQyNTggMTYuMDM5MSA4LjY0MjU4SDE0LjkyNjhWNy40NTExN0gxNi4wMzkxWk0xMi43MDEyIDguNjQyNThIMTAuNDc0NlY3LjQ1MTE3SDEyLjcwMTJWOC42NDI1OFpNMTEuOTk0MSA0VjUuMDkzNzVIMTAuODAxOEMxMC41NzI3IDUuMDkzNzYgMTAuMzg2NyA1LjI1NzU2IDEwLjM4NjcgNS40NTg5OFY2Ljg3MjA3SDkuMTQyNThWNS40NTg5OEM5LjE0MjU4IDQuNjUzMjYgOS44ODU1OCA0LjAwMDAxIDEwLjgwMTggNEgxMS45OTQxWk0yMC4zNDA4IDRDMjEuMjU2OSA0LjAwMDE0IDIxLjk5OSA0LjY1MzM0IDIxLjk5OSA1LjQ1ODk4VjYuODcyMDdIMjAuNzU0OVY1LjQ1ODk4QzIwLjc1NDkgNS4yNTc2NCAyMC41Njk3IDUuMDkzODkgMjAuMzQwOCA1LjA5Mzc1SDE5LjE0ODRWNEgyMC4zNDA4Wk0xNi43NjM3IDRWNS4wOTM3NUgxNC4zNzg5VjRIMTYuNzYzN1oiIGZpbGw9ImN1cnJlbnRDb2xvciIvPgo8L3N2Zz4K"),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.replicaset"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.replicaset",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *replicasetDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	replicasets := d.k8s.ReplicaSets()

	filteredReplicaSets := make([]*appsv1.ReplicaSet, 0, len(replicasets))
	for _, replicaset := range replicasets {
		if client.IsExcludedFromDiscovery(replicaset.ObjectMeta) {
			continue
		}
		filteredReplicaSets = append(filteredReplicaSets, replicaset)
	}

	nodes := d.k8s.Nodes()
	targets := make([]discovery_kit_api.Target, len(filteredReplicaSets))
	for i, replicaset := range filteredReplicaSets {
		attributes := map[string][]string{
			"k8s.namespace":      {replicaset.Namespace},
			"k8s.replicaset":     {replicaset.Name},
			"k8s.cluster-name":   {extconfig.Config.ClusterName},
			"k8s.distribution":   {d.k8s.Distribution},
			"k8s.container.name": {},
		}

		ownerReferences := client.OwnerReferences(d.k8s, &replicaset.ObjectMeta)
		for _, ownerRef := range ownerReferences.OwnerRefs {
			attributes[fmt.Sprintf("k8s.%v", ownerRef.Kind)] = []string{ownerRef.Name}
			attributes["k8s.workload-type"] = []string{ownerRef.Kind}
			attributes["k8s.workload-owner"] = []string{ownerRef.Name}
		}

		if replicaset.Spec.Replicas != nil {
			attributes["k8s.specification.replicas"] = []string{fmt.Sprintf("%d", *replicaset.Spec.Replicas)}
		}
		if replicaset.ObjectMeta.Annotations != nil {
			if value, ok := replicaset.ObjectMeta.Annotations["deployment.kubernetes.io/revision"]; ok {
				attributes["k8s.replicaset.revision"] = []string{value}
			}
		}
		extcommon.AddLabels(attributes, replicaset.ObjectMeta.Labels, "k8s.replicaset.label", "k8s.label")
		extcommon.AddNamespaceLabels(attributes, d.k8s, replicaset.Namespace)

		extcommon.MergeAttributes(
			attributes,
			extcommon.GetPodBasedAttributes("replicaset", replicaset.ObjectMeta, d.k8s.PodsByOwnerUid(replicaset.UID, replicaset.Namespace), nodes),
			extcommon.GetServiceNames(d.k8s.ServicesMatchingToPodLabels(replicaset.Namespace, replicaset.Spec.Template.Labels)),
		)

		for container := range replicaset.Spec.Template.Spec.Containers {
			attributes["k8s.container.name"] = append(attributes["k8s.container.name"], replicaset.Spec.Template.Spec.Containers[container].Name)
		}

		targets[i] = discovery_kit_api.Target{
			Id:         fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, replicaset.Namespace, replicaset.Name),
			TargetType: ReplicaSetTargetType,
			Label:      replicaset.Name,
			Attributes: attributes,
		}
	}
	targets = onlyHighestRevision(targets)
	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesReplicaSet), nil
}

func onlyHighestRevision(targets []discovery_kit_api.Target) []discovery_kit_api.Target {
	// Filter targets by workload-owner, workload-type, and namespace, keeping only the highest revision
	filteredTargets := make([]discovery_kit_api.Target, 0, len(targets))
	grouped := make(map[string]discovery_kit_api.Target)

	for _, t := range targets {
		owner := firstOrEmpty(t.Attributes["k8s.workload-owner"])
		typ := firstOrEmpty(t.Attributes["k8s.workload-type"])
		ns := firstOrEmpty(t.Attributes["k8s.namespace"])
		revStr := firstOrEmpty(t.Attributes["k8s.replicaset.revision"])

		if owner == "" || typ == "" || ns == "" || revStr == "" {
			// If any attribute is missing, add target as-is
			filteredTargets = append(filteredTargets, t)
			continue
		}

		key := fmt.Sprintf("%s|%s|%s", owner, typ, ns)
		rev, err := strconv.Atoi(revStr)
		if err != nil {
			filteredTargets = append(filteredTargets, t)
			continue
		}

		existing, exists := grouped[key]
		if !exists {
			grouped[key] = t
		} else {
			existingRev, err := strconv.Atoi(firstOrEmpty(existing.Attributes["k8s.replicaset.revision"]))
			if err != nil || rev > existingRev {
				grouped[key] = t
			}
		}
	}

	// Add grouped targets to filteredTargets
	for _, t := range grouped {
		filteredTargets = append(filteredTargets, t)
	}

	return filteredTargets
}

func firstOrEmpty(values []string) string {
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
