// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package client

import (
	"strings"

	"github.com/steadybit/extension-kit/extutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OwnerReference struct {
	Name string
	Kind string
}
type OwnerRefListWithResource struct {
	OwnerRefs  []OwnerReference
	Deployment *appsv1.Deployment
	Daemonset  *appsv1.DaemonSet
}

func (o OwnerRefListWithResource) ContainerSpec(containerName string) *corev1.Container {
	if o.Deployment != nil {
		return findContainerSpec(o.Deployment.Spec.Template.Spec.Containers, containerName)
	} else if o.Daemonset != nil {
		return findContainerSpec(o.Daemonset.Spec.Template.Spec.Containers, containerName)
	} else {
		return nil
	}
}

func findContainerSpec(specs []corev1.Container, containerName string) *corev1.Container {
	for _, spec := range specs {
		if spec.Name == containerName {
			return &spec
		}
	}
	return nil
}

func OwnerReferences(k8s *Client, meta *metav1.ObjectMeta) OwnerRefListWithResource {
	result := OwnerRefListWithResource{}
	recursivelyGetOwnerReferences(k8s, meta, &result)
	return result
}

func recursivelyGetOwnerReferences(k8s *Client, meta *metav1.ObjectMeta, result *OwnerRefListWithResource) {
	if meta.GetOwnerReferences() == nil {
		return
	}
	for _, ref := range meta.GetOwnerReferences() {
		ownerRef, ownerMeta, deployment, daemonset := getResource(k8s, ref.Kind, meta.Namespace, ref.Name)
		if ownerRef != nil {
			result.OwnerRefs = append(result.OwnerRefs, *ownerRef)
			if deployment != nil {
				result.Deployment = deployment
			}
			if daemonset != nil {
				result.Daemonset = daemonset
			}
			recursivelyGetOwnerReferences(k8s, ownerMeta, result)
		}
	}
}

func getResource(k8s *Client, kind string, namespace string, name string) (*OwnerReference, *metav1.ObjectMeta, *appsv1.Deployment, *appsv1.DaemonSet) {
	if strings.EqualFold("replicaset", kind) {
		replicaSet := k8s.ReplicaSetByNamespaceAndName(namespace, name)
		if replicaSet != nil {
			return extutil.Ptr(OwnerReference{Name: replicaSet.Name, Kind: strings.ToLower(kind)}), extutil.Ptr(replicaSet.ObjectMeta), nil, nil
		}
	} else if strings.EqualFold("daemonset", kind) {
		daemonSet := k8s.DaemonSetByNamespaceAndName(namespace, name)
		if daemonSet != nil {
			return extutil.Ptr(OwnerReference{Name: daemonSet.Name, Kind: strings.ToLower(kind)}), extutil.Ptr(daemonSet.ObjectMeta), nil, daemonSet
		}
	} else if strings.EqualFold("deployment", kind) {
		deployment := k8s.DeploymentByNamespaceAndName(namespace, name)
		if deployment != nil {
			return extutil.Ptr(OwnerReference{Name: deployment.Name, Kind: strings.ToLower(kind)}), extutil.Ptr(deployment.ObjectMeta), deployment, nil
		}
	} else if strings.EqualFold("statefulset", kind) {
		statefulset := k8s.StatefulSetByNamespaceAndName(namespace, name)
		if statefulset != nil {
			return extutil.Ptr(OwnerReference{Name: statefulset.Name, Kind: strings.ToLower(kind)}), extutil.Ptr(statefulset.ObjectMeta), nil, nil
		}
	}
	return nil, nil, nil, nil
}
