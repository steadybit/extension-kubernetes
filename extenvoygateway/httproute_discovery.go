// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
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
			CallInterval: new("30s"),
		},
	}
}

func (d *httpRouteDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       EnvoyGatewayHttpRouteTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Envoy HTTP Route", Other: "Envoy HTTP Routes"},
		Category: new("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     new(EnvoyGatewayIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: attrHttpRoute},
				{Attribute: "k8s.envoy-gateway.http-route.hostname"},
				{Attribute: "k8s.envoy-gateway.gateway"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{Attribute: attrHttpRoute, Direction: "ASC"},
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

// parseGatewayParentRef extracts a Gateway reference from an HTTPRoute parentRef entry, applying
// Gateway API defaulting: kind defaults to Gateway (non-Gateway kinds are skipped) and namespace
// defaults to the route's namespace. Returns false when the entry is not a usable Gateway reference.
func parseGatewayParentRef(ref any, routeNamespace string) (gatewayRef, bool) {
	refMap, ok := ref.(map[string]any)
	if !ok {
		return gatewayRef{}, false
	}
	if kind, ok := refMap["kind"].(string); ok && kind != "" && kind != "Gateway" {
		return gatewayRef{}, false
	}
	name, ok := refMap["name"].(string)
	if !ok || name == "" {
		return gatewayRef{}, false
	}
	namespace, _ := refMap["namespace"].(string)
	if namespace == "" {
		namespace = routeNamespace
	}
	return gatewayRef{namespace: namespace, name: name}, true
}

// resolveEnvoyGateways walks the route's parentRefs and returns the Gateways that belong to an Envoy
// Gateway GatewayClass, plus the resolved GatewayClass name.
func (d *httpRouteDiscovery) resolveEnvoyGateways(route *unstructured.Unstructured, gatewayToClass map[gatewayRef]string, envoyGatewayClasses map[string]bool) ([]gatewayRef, string) {
	parentRefs, found, err := unstructured.NestedSlice(route.Object, "spec", "parentRefs")
	if err != nil || !found {
		return nil, ""
	}

	var matching []gatewayRef
	for _, ref := range parentRefs {
		gw, ok := parseGatewayParentRef(ref, route.GetNamespace())
		if !ok {
			continue
		}
		if className, ok := gatewayToClass[gw]; ok && envoyGatewayClasses[className] {
			matching = append(matching, gw)
		}
	}
	// Sort and dedup so the resulting multi-valued attributes are stable across discovery cycles
	// (unstable ordering churns the platform's target store). Sorting the (namespace, name) pairs
	// together keeps the gateway / gateway.namespace attributes aligned, and makes the derived
	// gatewayclass deterministic when a route attaches to Envoy gateways of different classes.
	slices.SortFunc(matching, func(a, b gatewayRef) int {
		if c := strings.Compare(a.namespace, b.namespace); c != 0 {
			return c
		}
		return strings.Compare(a.name, b.name)
	})
	matching = slices.Compact(matching)

	var gatewayClass string
	if len(matching) > 0 {
		gatewayClass = gatewayToClass[matching[0]]
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
		attrHttpRoute:                    {name},
		"k8s.envoy-gateway.gatewayclass": {gatewayClass},
	}

	if hostnames, found, err := unstructured.NestedStringSlice(route.Object, "spec", "hostnames"); err == nil && found && len(hostnames) > 0 {
		attributes["k8s.envoy-gateway.http-route.hostname"] = sortDedup(hostnames)
	}

	var gatewayNames, gatewayNamespaces []string
	for _, gw := range gateways {
		gatewayNames = append(gatewayNames, gw.name)
		gatewayNamespaces = append(gatewayNamespaces, gw.namespace)
	}
	attributes["k8s.envoy-gateway.gateway"] = gatewayNames
	attributes["k8s.envoy-gateway.gateway.namespace"] = gatewayNamespaces

	if ruleNames := ruleNames(route); len(ruleNames) > 0 {
		attributes["k8s.envoy-gateway.http-route.rule"] = sortDedup(ruleNames)
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

// sortDedup returns a sorted, de-duplicated copy so multi-valued attributes stay stable across
// discovery cycles and don't churn the platform's target store.
func sortDedup(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	return slices.Compact(out)
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
