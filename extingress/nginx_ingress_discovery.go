/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extingress

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/steadybit/extension-kubernetes/v2/extnamespace"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

type nginxIngressDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.TargetDescriber = (*nginxIngressDiscovery)(nil)
)

func NewNginxIngressDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	discovery := &nginxIngressDiscovery{
		k8s: k8s,
	}

	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s,
		reflect.TypeOf(networkingv1.Ingress{}),
		reflect.TypeOf(networkingv1.IngressClass{}),
	)

	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, 5*time.Second),
	)
}

func (d *nginxIngressDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: NginxIngressTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *nginxIngressDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       NginxIngressTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "NGINX Ingress", Other: "NGINX Ingresses"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr("data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M12.0956%202L3%207.25V17.75L12.0956%2023L21.1912%2017.75V7.25L12.0956%202ZM17.3456%2016.5162C17.3456%2017.1331%2016.7804%2017.645%2016.0078%2017.645C15.4556%2017.645%2014.8256%2017.4219%2014.4319%2016.9362L9.18187%2010.6879V16.5154C9.18187%2017.1462%208.68312%2017.6441%208.06712%2017.6441H8.00062C7.36975%2017.6441%206.87187%2017.1191%206.87187%2016.5154V8.48375C6.87187%207.86687%207.42312%207.355%208.18437%207.355C8.74962%207.355%209.39187%207.57812%209.78562%208.06375L15.0094%2014.3121V8.48375C15.0094%207.85287%2015.5344%207.355%2016.1381%207.355H16.2037C16.8337%207.355%2017.3325%207.88%2017.3325%208.48375V16.5162H17.3456Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3C%2Fsvg%3E%0A"),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.ingress"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
				{Attribute: "k8s.ingress.class"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.ingress",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *nginxIngressDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	ingresses := d.k8s.Ingresses()

	nginxClasses, hasDefaultClass := d.k8s.GetNginxIngressClasses()
	log.Debug().Msgf("Found NGINX IngressClasses: %v, hasDefault: %v", nginxClasses, hasDefaultClass)

	filteredIngresses := make([]*networkingv1.Ingress, 0, len(ingresses))
	for _, ingress := range ingresses {
		if client.IsExcludedFromDiscovery(ingress.ObjectMeta) {
			continue
		}

		ingressClassName := d.getIngressClassName(ingress)
		usesNginxClass := d.isUsingNginxClass(ingressClassName, nginxClasses, hasDefaultClass)

		if usesNginxClass {
			filteredIngresses = append(filteredIngresses, ingress)
		}
	}

	targets := make([]discovery_kit_api.Target, len(filteredIngresses))

	for i, ingress := range filteredIngresses {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, ingress.Namespace, ingress.Name)
		attributes := map[string][]string{
			"k8s.namespace":    {ingress.Namespace},
			"k8s.ingress":      {ingress.Name},
			"k8s.cluster-name": {extconfig.Config.ClusterName},
			"k8s.distribution": {d.k8s.Distribution},
		}

		ingressClassName := d.getIngressClassName(ingress)
		if ingressClassName != "" {
			attributes["k8s.ingress.class"] = []string{ingressClassName}
			controller := d.k8s.GetIngressControllerByClassName(ingressClassName)
			if controller != "" {
				attributes["k8s.ingress.controller"] = []string{controller}
			}

			// Add NGINX controller namespace and pod information
			controllerNamespace := d.getNginxControllerInfo(ingressClassName)
			if controllerNamespace != "" {
				attributes["k8s.nginx.controller.namespace"] = []string{controllerNamespace}
			}
		}

		hosts := make([]string, 0)
		for _, rule := range ingress.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		if len(hosts) > 0 {
			attributes["k8s.ingress.hosts"] = hosts
		}

		for key, value := range ingress.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.ingress.label.%v", key)] = []string{value}
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		extnamespace.AddNamespaceLabels(d.k8s, ingress.Namespace, attributes)

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: NginxIngressTargetType,
			Label:      ingress.Name,
			Attributes: attributes,
		}
	}

	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesIngress), nil
}

func (d *nginxIngressDiscovery) getIngressClassName(ingress *networkingv1.Ingress) string {
	if ingress.Spec.IngressClassName != nil {
		return *ingress.Spec.IngressClassName
	}
	if ingress.ObjectMeta.Annotations != nil {
		if classAnnotation, ok := ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"]; ok {
			return classAnnotation
		}
	}
	return ""
}

func (d *nginxIngressDiscovery) isUsingNginxClass(className string, nginxClasses []string, hasDefaultClass bool) bool {
	if className != "" {
		for _, nginxClass := range nginxClasses {
			if className == nginxClass {
				return true
			}
		}
	} else if hasDefaultClass {
		return true
	}
	return false
}

// getNginxControllerInfo finds the NGINX controller namespace for the given ingress class
func (d *nginxIngressDiscovery) getNginxControllerInfo(ingressClassName string) string {
	// Search all namespaces for NGINX controllers
	allNamespaces := d.k8s.Namespaces()
	for _, ns := range allNamespaces {
		if d.hasNginxControllerForClass(ns.Name, ingressClassName) {
			log.Debug().Msgf("Found NGINX controller for IngressClass %s in namespace %s", ingressClassName, ns.Name)
			return ns.Name
		}
	}

	log.Debug().Msgf("No NGINX controller found for IngressClass %s", ingressClassName)
	return ""
}

// hasNginxControllerForClass checks if there's an NGINX controller in the namespace that handles the IngressClass
func (d *nginxIngressDiscovery) hasNginxControllerForClass(namespace string, ingressClassName string) bool {
	// NGINX controller label selectors for different variants
	labelSelectors := []map[string]string{
		// Community NGINX Ingress Controller (k8s.io/ingress-nginx)
		{"app.kubernetes.io/name": "ingress-nginx"},
		{"app.kubernetes.io/component": "controller", "app.kubernetes.io/name": "ingress-nginx"},
		
		// Enterprise NGINX Ingress Controller (nginx.org/ingress-controller) 
		{"app": "nginx-ingress"},
		{"app.kubernetes.io/name": "nginx-ingress"},
		
		// Legacy and custom installations
		{"k8s-app": "nginx-ingress-controller"},
		{"name": "nginx-ingress-controller"},
		{"app": "nginx-ingress-controller"},
		
		// Additional patterns for UBI and other variants
		{"app.kubernetes.io/component": "controller"},
		{"component": "nginx-ingress-controller"},
	}

	for _, selector := range labelSelectors {
		labelSelector := &metav1.LabelSelector{
			MatchLabels: selector,
		}
		pods := d.k8s.PodsByLabelSelector(labelSelector, namespace)

		for _, pod := range pods {
			// If we have a specific IngressClass name, check if this controller handles it
			if ingressClassName != "" && d.controllerHandlesClass(pod, ingressClassName) {
				return true
			}
			// If no specific IngressClass or controller doesn't specify class, assume it matches
			if ingressClassName == "" || !d.controllerHasClassArg(pod) {
				return true
			}
		}
	}

	return false
}

// controllerHandlesClass checks if the controller pod is configured for the specific IngressClass
func (d *nginxIngressDiscovery) controllerHandlesClass(pod *corev1.Pod, ingressClassName string) bool {
	if ingressClassName == "" {
		return false
	}
	
	for _, container := range pod.Spec.Containers {
		for _, arg := range container.Args {
			if strings.HasPrefix(arg, "--ingress-class="+ingressClassName) {
				return true
			}
		}
	}
	return false
}

// controllerHasClassArg checks if the controller has any --ingress-class argument
func (d *nginxIngressDiscovery) controllerHasClassArg(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		for _, arg := range container.Args {
			if strings.HasPrefix(arg, "--ingress-class=") {
				return true
			}
		}
	}
	return false
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
