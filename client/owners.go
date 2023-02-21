package client

import (
	"github.com/steadybit/extension-kit/extutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type OwnerReference struct {
	Name string
	Kind string
}
type ownerReferenceResult struct {
	ownerRefs []OwnerReference
}

func OwnerReferenceList(meta *metav1.ObjectMeta) []OwnerReference {
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

func getResource(kind string, namespace string, name string) (*OwnerReference, *metav1.ObjectMeta) {
	if strings.EqualFold("replicaset", kind) {
		replicaSet := K8S.ReplicaSetByNamespaceAndName(namespace, name)
		if replicaSet != nil {
			return extutil.Ptr(OwnerReference{Name: replicaSet.Name, Kind: strings.ToLower(kind)}), extutil.Ptr(replicaSet.ObjectMeta)
		}
	} else if strings.EqualFold("daemonset", kind) {
		daemonSet := K8S.DaemonSetByNamespaceAndName(namespace, name)
		if daemonSet != nil {
			return extutil.Ptr(OwnerReference{Name: daemonSet.Name, Kind: strings.ToLower(kind)}), extutil.Ptr(daemonSet.ObjectMeta)
		}
	} else if strings.EqualFold("deployment", kind) {
		deployment := K8S.DeploymentByNamespaceAndName(namespace, name)
		if deployment != nil {
			return extutil.Ptr(OwnerReference{Name: deployment.Name, Kind: strings.ToLower(kind)}), extutil.Ptr(deployment.ObjectMeta)
		}
	} else if strings.EqualFold("statefulset", kind) {
		statefulset := K8S.StatefulSetByNamespaceAndName(namespace, name)
		if statefulset != nil {
			return extutil.Ptr(OwnerReference{Name: statefulset.Name, Kind: strings.ToLower(kind)}), extutil.Ptr(statefulset.ObjectMeta)
		}
	}
	return nil, nil
}
