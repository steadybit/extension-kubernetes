package extcommon

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"golang.org/x/exp/maps"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func GetPodBasedAttributes(ownerType string, owner metav1.ObjectMeta, pods []*v1.Pod, nodes []*v1.Node) map[string][]string {
	attributes := map[string][]string{}
	if len(pods) > extconfig.Config.DiscoveryMaxPodCount {
		log.Warn().Msgf("%s %s/%s has more than %d pods. Not listing pods, containers and hosts for this %s", ownerType, owner.Namespace, owner.Name, extconfig.Config.DiscoveryMaxPodCount, ownerType)
		attributes["k8s.pod.name"] = []string{"too-many-pods"}
		attributes["k8s.container.id"] = []string{"too-many-pods"}
		attributes["k8s.container.id.stripped"] = []string{"too-many-pods"}
		attributes["host.hostname"] = []string{"too-many-pods"}
		attributes["host.domainname"] = []string{"too-many-pods"}
	} else if len(pods) > 0 {
		podNames := make([]string, 0, len(pods))
		var containerIds []string
		var containerIdsWithoutPrefix []string
		hostnames := make(map[string]bool)
		hostFQDNs := make(map[string]bool)
		for _, pod := range pods {
			podNames = append(podNames, pod.Name)
			for _, container := range pod.Status.ContainerStatuses {
				if container.ContainerID == "" {
					continue
				}
				containerIds = append(containerIds, container.ContainerID)
				containerIdsWithoutPrefix = append(containerIdsWithoutPrefix, strings.SplitAfter(container.ContainerID, "://")[1])
			}
			hostname, fqdns := GetNodeHostnameAndFQDNs(nodes, pod.Spec.NodeName)
			hostnames[hostname] = true
			for _, fqdn := range fqdns {
				hostFQDNs[fqdn] = true
			}
			AddNodeLabels(nodes, pod.Spec.NodeName, attributes)
		}
		attributes["k8s.pod.name"] = podNames
		if len(containerIds) > 0 {
			attributes["k8s.container.id"] = containerIds
		}
		if len(containerIdsWithoutPrefix) > 0 {
			attributes["k8s.container.id.stripped"] = containerIdsWithoutPrefix
		}
		if len(hostnames) > 0 {
			attributes["host.hostname"] = maps.Keys(hostnames)
		}
		if len(hostnames) > 0 {
			attributes["host.domainname"] = maps.Keys(hostFQDNs)
		}
	}
	return attributes
}

func GetServiceNames(services []*v1.Service) map[string][]string {
	attributes := map[string][]string{}
	if len(services) > 0 {
		var serviceNames = make([]string, 0, len(services))
		for _, service := range services {
			serviceNames = append(serviceNames, service.Name)
		}
		attributes["k8s.service.name"] = serviceNames
	}
	return attributes
}

func GetNodeHostnameAndFQDNs(nodes []*v1.Node, name string) (hostname string, fqdn []string) {
	for _, node := range nodes {
		if node.Name == name {
			return GetHostname(node), GetDomainnames(node)
		}
	}
	return "unknown", []string{"unknown"}
}

func GetDomainnames(node *v1.Node) []string {
	var names []string
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeInternalDNS {
			names = append(names, address.Address)
		}
	}
	if len(names) > 0 {
		return names
	} else {
		return []string{GetHostname(node)}
	}
}

func GetHostname(node *v1.Node) string {
	if hostname, ok := node.Labels["kubernetes.io/hostname"]; ok {
		return hostname
	}
	return node.Name
}
