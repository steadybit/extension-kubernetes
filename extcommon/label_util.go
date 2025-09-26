// Copyright 2025 steadybit GmbH. All rights reserved.

package extcommon

import (
	"fmt"
	"slices"

	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	corev1 "k8s.io/api/core/v1"
)

func AddFilteredLabels(attributes map[string][]string, filter func(key string) bool, labels map[string]string, prefixes ...string) map[string][]string {
	for _, prefix := range prefixes {
		allKeys := make([]string, 0, len(labels))

		for key, value := range labels {
			if filter(key) {
				allKeys = append(allKeys, key)
				mergeAttributeValues(attributes, fmt.Sprintf("%s.%s", prefix, key), value)
			}
		}

		if len(allKeys) > 0 {
			mergeAttributeValues(attributes, prefix, allKeys...)
		}
	}

	return attributes
}

func AddLabels(attributes map[string][]string, labels map[string]string, prefixes ...string) map[string][]string {
	return AddFilteredLabels(attributes, func(key string) bool {
		return !slices.Contains(extconfig.Config.LabelFilter, key)
	}, labels, prefixes...)
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
			return AddFilteredLabels(attributes, func(key string) bool {
				return slices.Contains(nodeLabelFilter, key)
			}, node.ObjectMeta.Labels, "k8s.node.label", "k8s.label")
		}
	}
	return attributes
}

func AddNamespaceLabels(attributes map[string][]string, k8s *client.Client, namespace string) map[string][]string {
	if k8s.Permissions().CanReadNamespaces() {
		for _, ns := range k8s.Namespaces() {
			if ns.Name == namespace {
				return AddLabels(attributes, ns.ObjectMeta.Labels, "k8s.namespace.label", "k8s.label")
			}
		}
	}
	return attributes
}
