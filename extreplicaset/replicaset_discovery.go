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
	"github.com/steadybit/extension-kubernetes/v2/extnamespace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
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
		Icon:     extutil.Ptr("data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A"),
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

	targets := make([]discovery_kit_api.Target, len(filteredReplicaSets))

	nodes := d.k8s.Nodes()
	for i, replicaset := range filteredReplicaSets {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, replicaset.Namespace, replicaset.Name)
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
		for key, value := range replicaset.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.replicaset.label.%v", key)] = []string{value}
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}
		if replicaset.ObjectMeta.Annotations != nil {
			if value, ok := replicaset.ObjectMeta.Annotations["deployment.kubernetes.io/revision"]; ok {
				attributes["k8s.replicaset.revision"] = []string{value}
			}
		}
		extnamespace.AddNamespaceLabels(d.k8s, replicaset.Namespace, attributes)

		for key, value := range extcommon.GetPodBasedAttributes("replicaset", replicaset.ObjectMeta, d.k8s.PodsByLabelSelector(replicaset.Spec.Selector, replicaset.Namespace), nodes) {
			attributes[key] = value
		}
		for key, value := range extcommon.GetServiceNames(d.k8s.ServicesMatchingToPodLabels(replicaset.Namespace, replicaset.Spec.Template.Labels)) {
			attributes[key] = value
		}

		for container := range replicaset.Spec.Template.Spec.Containers {
			attributes["k8s.container.name"] = append(
				attributes["k8s.container.name"],
				replicaset.Spec.Template.Spec.Containers[container].Name,
			)
		}

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
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
