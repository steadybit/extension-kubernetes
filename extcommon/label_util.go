// Copyright 2025 steadybit GmbH. All rights reserved.

package extcommon

import (
	"fmt"
	"slices"

	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	corev1 "k8s.io/api/core/v1"
)

func AddFilteredLabels(filter func(key string) bool, labels map[string]string, attributes map[string][]string, prefixes ...string) map[string][]string {
	for _, prefix := range prefixes {
		allKeys := make([]string, 0, len(labels))

		for key, value := range labels {
			if filter(key) {
				allKeys = appendIfNotPresent(allKeys, key)
				attributeKey := fmt.Sprintf("%s.%s", prefix, key)
				attributes[attributeKey] = appendIfNotPresent(attributes[attributeKey], value)
			}
		}

		if len(allKeys) > 0 {
			attributes[prefix] = appendIfNotPresent(attributes[prefix], allKeys...)
		}
	}

	return attributes

}

func AddLabels(labels map[string]string, attributes map[string][]string, prefixes ...string) map[string][]string {
	return AddFilteredLabels(func(key string) bool {
		return !slices.Contains(extconfig.Config.LabelFilter, key)
	}, labels, attributes, prefixes...)
}

var nodeLabelFilter = []string{
	"topology.kubernetes.io/region",
	"topology.kubernetes.io/zone",
	"kubernetes.io/arch",
	"kubernetes.io/os",
	"node.kubernetes.io/instance-type",
}

func AddNodeLabels(nodes []*corev1.Node, nodeName string, attributes map[string][]string) map[string][]string {
	for _, node := range nodes {
		if node.Name == nodeName {
			return AddFilteredLabels(func(key string) bool {
				return slices.Contains(nodeLabelFilter, key)
			}, node.ObjectMeta.Labels, attributes, "k8s.node.label", "k8s.label")
		}
	}
	return attributes
}

func AddNamespaceLabels(k8s *client.Client, namespace string, attributes map[string][]string) map[string][]string {
	if k8s.Permissions().CanReadNamespaces() {
		for _, ns := range k8s.Namespaces() {
			if ns.Name == namespace {
				return AddLabels(ns.ObjectMeta.Labels, attributes, "k8s.namespace.label", "k8s.label")
			}
		}
	}
	return attributes
}

func appendIfNotPresent(slice []string, elements ...string) []string {
	for _, e := range elements {
		if !slices.Contains(slice, e) {
			slice = append(slice, e)
			slices.Sort(append(slice, e))
		}
	}
	return slice
}
