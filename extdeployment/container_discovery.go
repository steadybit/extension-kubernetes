// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"fmt"
	discovery_kit_api "github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

func RegisterContainerDiscoveryHandlers() {
	exthttp.RegisterHttpHandler("/container/discovery", exthttp.GetterAsHandler(getContainerDiscoveryDescription))
	exthttp.RegisterHttpHandler("/container/discovery/target-description", exthttp.GetterAsHandler(getContainerTargetDescription))
	exthttp.RegisterHttpHandler("/container/discovery/discovered-targets", getDiscoveredContainer)
}

func getContainerDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         containerTargetType,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         "/container/discovery/discovered-targets",
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func getContainerTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       containerTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes container", Other: "Kubernetes container"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  "1.0.0-SNAPSHOT",
		Icon:     extutil.Ptr(containerIcon),
		EnrichmentRules: extutil.Ptr([]discovery_kit_api.TargetEnrichmentRule{
			{
				Src: discovery_kit_api.SourceOrDestination{
					Type: containerTargetType,
					Selector: map[string]string{
						"container.id": "${dest.container.id}",
					},
				},
				Dest: discovery_kit_api.SourceOrDestination{
					Type: "container",
					Selector: map[string]string{
						"container.id": "${src.container.id}",
					},
				},
				Attributes: []discovery_kit_api.Attribute{
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.cluster-name",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.distribution",
					}, {
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.namespace",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.container.name",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.container.ready",
					},
					{
						//TODO --> Do we need an explicit "overwrite"?
						AggregationType: discovery_kit_api.Any,
						Name:            "container.image",
					},
					{
						AggregationType: discovery_kit_api.All,
						Name:            "k8s.service.name",
					},
					{
						AggregationType: discovery_kit_api.All,
						Name:            "k8s.service.namespace",
					},
					//TODO POD-Labels --> Do we need wildcards?
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.replicaset",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.daemonset",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.deployment",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.statefulset",
					},
				},
			}, {
				Src: discovery_kit_api.SourceOrDestination{
					Type: containerTargetType,
					Selector: map[string]string{
						"host.hostname": "${dest.host.hostname}",
					},
				},
				Dest: discovery_kit_api.SourceOrDestination{
					Type: "host",
					Selector: map[string]string{
						"host.hostname": "${src.host.hostname}",
					},
				},
				Attributes: []discovery_kit_api.Attribute{
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.cluster-name",
					},
					{
						AggregationType: discovery_kit_api.Any,
						Name:            "k8s.distribution",
					},
					//TODO Labels  --> Do we need wildcards?
				},
			},
		}),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.container.name"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
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

func getDiscoveredContainer(w http.ResponseWriter, r *http.Request, _ []byte) {
	var targets []discovery_kit_api.Target

	for _, pod := range client.K8S.Pods() {
		podMetadata := pod.ObjectMeta
		ownerReferenceList := getOwnerReferenceList(&podMetadata)
		fmt.Printf("Pod: %s \n ReferenceList:%+v\n", pod.Name, ownerReferenceList)
	}

	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}

type ownerReference struct {
	name string
	kind string
}
type ownerReferenceResult struct {
	ownerRefs []ownerReference
}

func getOwnerReferenceList(meta *metav1.ObjectMeta) []ownerReference {
	result := ownerReferenceResult{}
	recursivelyGetOwnerReferences(meta, &result)
	return result.ownerRefs
}

func recursivelyGetOwnerReferences(meta *metav1.ObjectMeta, result *ownerReferenceResult) {
	if meta.GetOwnerReferences() == nil {
		return
	}
	for _, ref := range meta.GetOwnerReferences() {
		ownerRef, ownerMeta := getResource(ref.Kind, meta.Namespace, ref.Name)
		if ownerRef != nil {
			result.ownerRefs = append(result.ownerRefs, *ownerRef)
			recursivelyGetOwnerReferences(ownerMeta, result)
		}
	}
}

func getResource(kind string, namespace string, name string) (*ownerReference, *metav1.ObjectMeta) {
	if strings.EqualFold("replicaset", kind) {
		replicaSet := client.K8S.ReplicaSetByNamespaceAndName(namespace, name)
		if replicaSet != nil {
			return extutil.Ptr(ownerReference{name: replicaSet.Name, kind: kind}), extutil.Ptr(replicaSet.ObjectMeta)
		}
	} else if strings.EqualFold("daemonset", kind) {
		daemonSet := client.K8S.DaemonSetByNamespaceAndName(namespace, name)
		if daemonSet != nil {
			return extutil.Ptr(ownerReference{name: daemonSet.Name, kind: kind}), extutil.Ptr(daemonSet.ObjectMeta)
		}
	} else if strings.EqualFold("deployment", kind) {
		deployment := client.K8S.DeploymentByNamespaceAndName(namespace, name)
		if deployment != nil {
			return extutil.Ptr(ownerReference{name: deployment.Name, kind: kind}), extutil.Ptr(deployment.ObjectMeta)
		}
	} else if strings.EqualFold("statefulset", kind) {
		statefulset := client.K8S.StatefulSetByNamespaceAndName(namespace, name)
		if statefulset != nil {
			return extutil.Ptr(ownerReference{name: statefulset.Name, kind: kind}), extutil.Ptr(statefulset.ObjectMeta)
		}
	}
	return nil, nil
}
