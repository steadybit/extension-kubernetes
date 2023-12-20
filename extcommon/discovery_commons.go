package extcommon

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func GetPodBasedAttributes(client *client.Client, objectMeta *metav1.ObjectMeta, labelSelector *metav1.LabelSelector) map[string][]string {
	attributes := map[string][]string{}
	pods := client.PodsByLabelSelector(labelSelector, objectMeta.Namespace)
	if len(pods) > extconfig.Config.DiscoveryMaxPodCount {
		log.Warn().Msgf("%s/%s has more than %d pods. Skip listing pods, containers and hosts.", objectMeta.Namespace, objectMeta.Name, extconfig.Config.DiscoveryMaxPodCount)
		attributes["k8s.pod.name"] = []string{"too-many-pods"}
		attributes["k8s.container.id"] = []string{"too-many-pods"}
		attributes["k8s.container.id.stripped"] = []string{"too-many-pods"}
		attributes["host.hostname"] = []string{"too-many-pods"}
	} else if len(pods) > 0 {
		podNames := make([]string, len(pods))
		var containerIds []string
		var containerIdsWithoutPrefix []string
		hostnames := make(map[string]bool)
		for podIndex, pod := range pods {
			podNames[podIndex] = pod.Name
			for _, container := range pod.Status.ContainerStatuses {
				if container.ContainerID == "" {
					continue
				}
				containerIds = append(containerIds, container.ContainerID)
				containerIdsWithoutPrefix = append(containerIdsWithoutPrefix, strings.SplitAfter(container.ContainerID, "://")[1])
			}
			hostnames[pod.Spec.NodeName] = true
		}
		attributes["k8s.pod.name"] = podNames
		if len(containerIds) > 0 {
			attributes["k8s.container.id"] = containerIds
		}
		if len(containerIdsWithoutPrefix) > 0 {
			attributes["k8s.container.id.stripped"] = containerIdsWithoutPrefix
		}
		if len(hostnames) > 0 {
			attributes["host.hostname"] = make([]string, 0, len(hostnames))
			for k := range hostnames {
				attributes["host.hostname"] = append(attributes["host.hostname"], k)
			}
		}
	}
	return attributes
}

func GetServiceNames(client *client.Client, namespace *string, template *v1.PodTemplateSpec) map[string][]string {
	attributes := map[string][]string{}
	services := client.ServicesMatchingToPodLabels(*namespace, template.ObjectMeta.Labels)
	if len(services) > 0 {
		var serviceNames = make([]string, 0, len(services))
		for _, service := range services {
			serviceNames = append(serviceNames, service.Name)
		}
		attributes["k8s.service.name"] = serviceNames
	}
	return attributes
}
