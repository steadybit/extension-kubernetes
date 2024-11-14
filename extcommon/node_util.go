package extcommon

import (
	"fmt"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
)

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
			nodeMetadata := node.ObjectMeta
			for key, value := range nodeMetadata.Labels {
				if slices.Contains(nodeLabelFilter, key) {
					attributeKey := fmt.Sprintf("k8s.label.%v", key)
					if _, ok := attributes[attributeKey]; ok {
						if !slices.Contains(attributes[attributeKey], value) {
							attributes[attributeKey] = append(attributes[attributeKey], value)
						}
					} else {
						attributes[attributeKey] = []string{value}
					}
				}
			}
		}
	}
	return attributes
}
