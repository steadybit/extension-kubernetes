// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"context"
	"fmt"
	"regexp"
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
	"reflect"
)

type ingressDiscovery struct {
	k8s                   *client.Client
	ingressClassNameRegex *regexp.Regexp
}

var (
	_ discovery_kit_sdk.TargetDescriber = (*ingressDiscovery)(nil)
)

func NewIngressDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	var ingressClassNameRegex *regexp.Regexp
	if extconfig.Config.IngressClassNameRegex != "" {
		var err error
		ingressClassNameRegex, err = regexp.Compile(extconfig.Config.IngressClassNameRegex)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to compile ingressClassName regex '%s', will not filter by ingressClassName", extconfig.Config.IngressClassNameRegex)
		}
	}

	discovery := &ingressDiscovery{
		k8s:                   k8s,
		ingressClassNameRegex: ingressClassNameRegex,
	}

	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s,
		reflect.TypeOf(networkingv1.Ingress{}),
	)

	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, 5*time.Second),
	)
}

func (d *ingressDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: IngressTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *ingressDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       IngressTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "HAProxy Ingress", Other: "HAProxy Ingresses"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr("data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M19.5%2013.5V5.5H12.5V3.5H21.5V13.5H19.5ZM4.5%2010.5V18.5H11.5V20.5H2.5V10.5H4.5ZM10.76%204.59L9.47%205.88L12.59%209H8V11H12.59L9.47%2014.12L10.76%2015.41L15.76%2010.41L10.76%204.59Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E"),
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

func (d *ingressDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	ingresses := d.k8s.Ingresses()

	filteredIngresses := make([]*networkingv1.Ingress, 0, len(ingresses))
	for _, ingress := range ingresses {
		if client.IsExcludedFromDiscovery(ingress.ObjectMeta) {
			continue
		}

		// Filter by ingressClassName regex if configured
		if d.ingressClassNameRegex != nil {
			ingressClassName := ""

			// First, check for the spec.ingressClassName field
			if ingress.Spec.IngressClassName != nil {
				ingressClassName = *ingress.Spec.IngressClassName
			} else if ingress.ObjectMeta.Annotations != nil {
				// If spec.ingressClassName is nil, fall back to the annotation
				if classAnnotation, ok := ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"]; ok {
					ingressClassName = classAnnotation
				}
			}

			// If we have a class name (from either source), check the regex
			if ingressClassName != "" {
				if !d.ingressClassNameRegex.MatchString(ingressClassName) {
					continue
				}
			} else {
				// No class name found, skip if the regex is configured
				continue
			}
		}

		filteredIngresses = append(filteredIngresses, ingress)
	}

	targets := make([]discovery_kit_api.Target, len(filteredIngresses))

	for i, ingress := range filteredIngresses {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, ingress.Namespace, ingress.Name)
		attributes := map[string][]string{
			"k8s.namespace":     {ingress.Namespace},
			"k8s.ingress":       {ingress.Name},
			"k8s.cluster-name":  {extconfig.Config.ClusterName},
			"k8s.distribution":  {d.k8s.Distribution},
		}

		// Add ingressClassName from spec or annotation
		ingressClassName := ""
		if ingress.Spec.IngressClassName != nil {
			ingressClassName = *ingress.Spec.IngressClassName
		} else if ingress.ObjectMeta.Annotations != nil {
			if classAnnotation, ok := ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"]; ok {
				ingressClassName = classAnnotation
			}
		}

		if ingressClassName != "" {
			attributes["k8s.ingress.class"] = []string{ingressClassName}
		}

		// Add all ingress rules hosts
		hosts := make([]string, 0)
		for _, rule := range ingress.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		if len(hosts) > 0 {
			attributes["k8s.ingress.hosts"] = hosts
		}

		// Add TLS secret names if available
		tlsSecrets := make([]string, 0)
		for _, tls := range ingress.Spec.TLS {
			if tls.SecretName != "" {
				tlsSecrets = append(tlsSecrets, tls.SecretName)
			}
		}
		if len(tlsSecrets) > 0 {
			attributes["k8s.ingress.tls.secrets"] = tlsSecrets
		}

		// Add labels
		for key, value := range ingress.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.ingress.label.%v", key)] = []string{value}
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		// Add namespace labels
		extnamespace.AddNamespaceLabels(d.k8s, ingress.Namespace, attributes)

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: IngressTargetType,
			Label:      ingress.Name,
			Attributes: attributes,
		}
	}

	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesIngress), nil
}

