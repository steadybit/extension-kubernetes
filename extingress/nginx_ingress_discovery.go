/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extingress

import (
	"context"
	"fmt"
	"reflect"
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
	networkingv1 "k8s.io/api/networking/v1"
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
		Icon:     extutil.Ptr("data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20version%3D%221.1%22%20viewBox%3D%220%200%2032%2035.2%22%3E%3Cpath%20d%3D%22m16%200-16%208.8v17.6l16%208.8%2016-8.8v-17.6zm10.9%2021.7c0%200.2-0.2%200.3-0.3%200.4l-2.3%201.3c-0.2%200.1-0.3%200.1-0.5%200l-2.4-1.4c-0.2-0.1-0.2-0.3-0.2-0.4v-2.7c0-0.2%200.2-0.3%200.3-0.4l2.3-1.3c0.2-0.1%200.3-0.1%200.5%200l2.4%201.4c0.2%200.1%200.2%200.3%200.2%200.4zm-10.9%209.1c-0.3%200-0.5-0.1-0.8-0.2l-5.3-3c-0.2-0.1-0.1-0.3%200.1-0.3%200.8-0.2%201.3-0.3%201.6-0.4%200.3-0.2%200.6-0.3%200.8-0.5l5.4%203.1c0.2%200.1%200.4%200.1%200.6%200l5.3-3c0.3%200.2%200.5%200.3%200.8%200.5%200.3%200.1%200.9%200.3%201.6%200.4%200.2%200%200.3%200.3%200.1%200.3l-5.3%203c-0.3%200.1-0.5%200.2-0.8%200.2zm0-4.9c-0.3%200-0.5-0.1-0.8-0.2l-5.3-3c-0.2-0.1-0.1-0.3%200.1-0.3%200.8-0.2%201.3-0.3%201.6-0.4%200.3-0.2%200.6-0.3%200.8-0.5l5.4%203.1c0.2%200.1%200.4%200.1%200.6%200l5.3-3c0.3%200.2%200.5%200.3%200.8%200.5%200.3%200.1%200.9%200.3%201.6%200.4%200.2%200%200.3%200.3%200.1%200.3l-5.3%203c-0.3%200.1-0.5%200.2-0.8%200.2zm0-4.9c-0.3%200-0.5-0.1-0.8-0.2l-5.4-3.1c-0.1-0.1-0.2-0.2-0.2-0.3v-6.1c0-0.2%200.2-0.3%200.3-0.2%200.7%200.4%201.2%200.6%201.6%200.8%200.3%200.1%200.6%200.3%200.8%200.5v5.2c0%200.1%200.1%200.2%200.2%200.3l4.5%202.6c0.2%200.1%200.4%200.1%200.6%200l4.5-2.6c0.1-0.1%200.2-0.2%200.2-0.3v-5.2c0.2-0.2%200.5-0.4%200.8-0.5%200.3-0.2%200.9-0.4%201.6-0.8%200.2-0.1%200.3%200.1%200.3%200.2v6.1c0%200.1-0.1%200.3-0.2%200.3l-5.4%203.1c-0.3%200.1-0.5%200.2-0.8%200.2zm-10.9-7.3c0-0.2%200.2-0.3%200.3-0.4l2.3-1.3c0.2-0.1%200.3-0.1%200.5%200l2.4%201.4c0.2%200.1%200.2%200.3%200.2%200.4v2.7c0%200.2-0.2%200.3-0.3%200.4l-2.3%201.3c-0.2%200.1-0.3%200.1-0.5%200l-2.4-1.4c-0.2-0.1-0.2-0.3-0.2-0.4z%22%20fill%3D%22%23009639%22%2F%3E%3C%2Fsvg%3E"),
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
