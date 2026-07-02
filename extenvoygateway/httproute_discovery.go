// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type httpRouteDiscovery struct {
	k8s *client.Client
}

var _ discovery_kit_sdk.TargetDescriber = (*httpRouteDiscovery)(nil)

func NewHttpRouteDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	discovery := &httpRouteDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s,
		reflect.TypeFor[unstructured.Unstructured](),
	)
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, 5*time.Second),
	)
}

func (d *httpRouteDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: EnvoyGatewayHttpRouteTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *httpRouteDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       EnvoyGatewayHttpRouteTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Envoy Gateway HTTP Route", Other: "Envoy Gateway HTTP Routes"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(EnvoyGatewayIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.envoy-gateway.http-route"},
				{Attribute: "k8s.envoy-gateway.http-route.hostname"},
				{Attribute: "k8s.envoy-gateway.gateway"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{Attribute: "k8s.envoy-gateway.http-route", Direction: "ASC"},
			},
		},
	}
}

// gatewayRef identifies a Gateway referenced by an HTTPRoute parentRef.
type gatewayRef struct {
	namespace string
	name      string
}

func (d *httpRouteDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	envoyGatewayClasses := d.envoyManagedGatewayClasses()
	if len(envoyGatewayClasses) == 0 {
		return []discovery_kit_api.Target{}, nil
	}
	gatewayToClass := d.gatewayToGatewayClass()

	var targets []discovery_kit_api.Target
	for _, route := range d.k8s.HTTPRoutes() {
		if client.IsExcludedFromDiscovery(objectMetaFromUnstructured(route)) {
			continue
		}

		matchingGateways, gatewayClass := d.resolveEnvoyGateways(route, gatewayToClass, envoyGatewayClasses)
		if len(matchingGateways) == 0 {
			continue
		}

		targets = append(targets, d.toTarget(route, matchingGateways, gatewayClass))
	}

	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesEnvoyGateway), nil
}

// envoyManagedGatewayClasses returns the set of GatewayClass names whose controllerName is Envoy Gateway's.
func (d *httpRouteDiscovery) envoyManagedGatewayClasses() map[string]bool {
	result := map[string]bool{}
	for _, gc := range d.k8s.GatewayClasses() {
		controllerName, found, err := unstructured.NestedString(gc.Object, "spec", "controllerName")
		if err == nil && found && controllerName == envoyGatewayControllerName {
			result[gc.GetName()] = true
		}
	}
	return result
}

// gatewayToGatewayClass maps each Gateway (namespace/name) to its gatewayClassName.
func (d *httpRouteDiscovery) gatewayToGatewayClass() map[gatewayRef]string {
	result := map[gatewayRef]string{}
	for _, gw := range d.k8s.Gateways() {
		className, found, err := unstructured.NestedString(gw.Object, "spec", "gatewayClassName")
		if err == nil && found {
			result[gatewayRef{namespace: gw.GetNamespace(), name: gw.GetName()}] = className
		}
	}
	return result
}

// resolveEnvoyGateways walks the route's parentRefs and returns the Gateways that belong to an Envoy
// Gateway GatewayClass, plus the resolved GatewayClass name.
func (d *httpRouteDiscovery) resolveEnvoyGateways(route *unstructured.Unstructured, gatewayToClass map[gatewayRef]string, envoyGatewayClasses map[string]bool) ([]gatewayRef, string) {
	parentRefs, found, err := unstructured.NestedSlice(route.Object, "spec", "parentRefs")
	if err != nil || !found {
		return nil, ""
	}

	var matching []gatewayRef
	var gatewayClass string
	for _, ref := range parentRefs {
		refMap, ok := ref.(map[string]any)
		if !ok {
			continue
		}
		// parentRef kind defaults to Gateway; group defaults to gateway.networking.k8s.io.
		if kind, ok := refMap["kind"].(string); ok && kind != "" && kind != "Gateway" {
			continue
		}
		name, ok := refMap["name"].(string)
		if !ok || name == "" {
			continue
		}
		namespace, _ := refMap["namespace"].(string)
		if namespace == "" {
			namespace = route.GetNamespace()
		}

		gw := gatewayRef{namespace: namespace, name: name}
		className, ok := gatewayToClass[gw]
		if !ok || !envoyGatewayClasses[className] {
			continue
		}
		matching = append(matching, gw)
		gatewayClass = className
	}
	return matching, gatewayClass
}

func (d *httpRouteDiscovery) toTarget(route *unstructured.Unstructured, gateways []gatewayRef, gatewayClass string) discovery_kit_api.Target {
	namespace := route.GetNamespace()
	name := route.GetName()

	attributes := map[string][]string{
		"k8s.namespace":                  {namespace},
		"k8s.cluster-name":               {extconfig.Config.ClusterName},
		"k8s.distribution":               {d.k8s.Distribution},
		"k8s.envoy-gateway.http-route":   {name},
		"k8s.envoy-gateway.gatewayclass": {gatewayClass},
	}

	if hostnames, found, err := unstructured.NestedStringSlice(route.Object, "spec", "hostnames"); err == nil && found && len(hostnames) > 0 {
		attributes["k8s.envoy-gateway.http-route.hostname"] = hostnames
	}

	var gatewayNames, gatewayNamespaces []string
	for _, gw := range gateways {
		gatewayNames = append(gatewayNames, gw.name)
		gatewayNamespaces = append(gatewayNamespaces, gw.namespace)
	}
	attributes["k8s.envoy-gateway.gateway"] = gatewayNames
	attributes["k8s.envoy-gateway.gateway.namespace"] = gatewayNamespaces

	if ruleNames := ruleNames(route); len(ruleNames) > 0 {
		attributes["k8s.envoy-gateway.http-route.rule"] = ruleNames
	}

	for key, value := range route.GetLabels() {
		if !slices.Contains(extconfig.Config.LabelFilter, key) {
			attributes[fmt.Sprintf("k8s.envoy-gateway.http-route.label.%v", key)] = []string{value}
			attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
		}
	}

	extcommon.AddNamespaceLabels(attributes, d.k8s, namespace)

	return discovery_kit_api.Target{
		Id:         fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, namespace, name),
		TargetType: EnvoyGatewayHttpRouteTargetType,
		Label:      name,
		Attributes: attributes,
	}
}

// ruleNames returns the names of named route rules (eligible for sectionName targeting).
func ruleNames(route *unstructured.Unstructured) []string {
	rules, found, err := unstructured.NestedSlice(route.Object, "spec", "rules")
	if err != nil || !found {
		return nil
	}
	var names []string
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if name, ok := ruleMap["name"].(string); ok && name != "" {
			names = append(names, name)
		}
	}
	return names
}
