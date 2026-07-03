// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// EnvoyGatewayHttpRouteTargetType is the discovery target type for HTTPRoutes managed by Envoy Gateway.
	EnvoyGatewayHttpRouteTargetType = "com.steadybit.extension_kubernetes.envoy-gateway-http-route"

	// attrHttpRoute is the discovery attribute holding the HTTPRoute name.
	attrHttpRoute = "k8s.envoy-gateway.http-route"

	DelayActionId        = "com.steadybit.extension_kubernetes.envoy-gateway-http-route-delay"
	StatusActionId       = "com.steadybit.extension_kubernetes.envoy-gateway-http-route-status"
	ResponseBodyActionId = "com.steadybit.extension_kubernetes.envoy-gateway-http-route-response-body"

	// envoyGatewayControllerName identifies GatewayClasses managed by Envoy Gateway.
	envoyGatewayControllerName = "gateway.envoyproxy.io/gatewayclass-controller"

	gatewayAPIGroup   = "gateway.networking.k8s.io"
	httpRouteKind     = "HTTPRoute"
	btpAPIVersion     = "gateway.envoyproxy.io/v1alpha1"
	btpKind           = "BackendTrafficPolicy"
	managedByLabelKey = "steadybit.com/managed-by"
	managedByValue    = "extension-kubernetes"
	executionLabelKey = "steadybit.com/execution-id"

	// EnvoyGatewayIcon is the official Envoy Gateway logo
	// (envoyproxy/gateway: site/assets/icons/logo.svg).
	EnvoyGatewayIcon = "data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20role%3D%22img%22%20viewBox%3D%22-4.21%2049.54%20439.92%20332.67%22%3E%3Cstyle%3Esvg%20%7Benable-background%3Anew%200%200%20432%20432%7D%3C%2Fstyle%3E%3Cpath%20fill%3D%22%23b31aab%22%20d%3D%22M109.8%20210.6l.6%2025.4%2026.8%2016.6-.6-25.4zm65.4%20105.8l-.6-24.9-23.5-14.6c-.3-.2-.7-.5-1-.7l.6%2025%2024.5%2015.2zM91.5%20350l-61.3-38-1.5-63.7%2030.1-13-.6-25.5-48%2020.7c-3.7%201.6-5.9%205-5.8%208.9l1.8%2076.5c.1%203.9%202.5%207.8%206.3%2010.2L86%20371.7c3.4%202.1%207.6%202.7%2011%201.6.4-.1.7-.2%201-.4l45.1-19.4-24.5-15.2L91.5%20350z%22%2F%3E%3Cpath%20fill%3D%22%23d163ce%22%20d%3D%22M289.6%20209.1c-.1-4.6-2.9-9.1-7.3-11.9L193%20141.9l-2.8%201.2.6%2026.8%2070.7%2043.8%201.7%2071.6%2027%2016.7%201.5-.6-2.1-92.3zM182.7%20334.8l-82.9-51.4-2-86.3%2037.8-16.3-.7-29.7-58.7%2025.3c-4.3%201.9-6.9%205.8-6.8%2010.4L71.7%20288c.1%204.6%202.9%209.1%207.3%2011.8l97.2%2060.3c4%202.5%208.8%203.1%2012.9%201.9.4-.1.8-.3%201.2-.5l57.4-24.7-28.6-17.7-36.4%2015.7z%22%2F%3E%3Cpath%20fill%3D%22%23e13eaf%22%20d%3D%22M415.9%20138.3L291.3%2061c-4.6-2.8-10.1-3.6-14.8-2.1-.5.1-.9.3-1.4.5l-121.6%2052.4c-4.9%202.1-7.9%206.6-7.8%2011.9l3.1%20129.6c.1%205.3%203.3%2010.4%208.4%2013.5L281.8%20344c4.6%202.8%2010.1%203.6%2014.7%202.1.5-.1.9-.3%201.4-.5l121.6-52.4c4.9-2.1%207.9-6.7%207.8-11.9l-3-129.6c-.1-5.1-3.3-10.3-8.4-13.4zM289.3%20315.2L181%20248.1l-2.7-112.7%20105.6-45.5L392.2%20157l2.7%20112.7-105.6%2045.5z%22%2F%3E%3C%2Fsvg%3E"
)

// buildBackendTrafficPolicy builds an unstructured Envoy Gateway BackendTrafficPolicy object.
// faultSpec is merged into spec alongside the targetRefs. sectionName is optional (empty = whole route).
func buildBackendTrafficPolicy(namespace, name, executionId, routeName, sectionName string, faultSpec map[string]any) *unstructured.Unstructured {
	targetRef := map[string]any{
		"group": gatewayAPIGroup,
		"kind":  httpRouteKind,
		"name":  routeName,
	}
	if sectionName != "" {
		targetRef["sectionName"] = sectionName
	}

	spec := map[string]any{
		"targetRefs": []any{targetRef},
	}
	for k, v := range faultSpec {
		spec[k] = v
	}

	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": btpAPIVersion,
			"kind":       btpKind,
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]any{
					managedByLabelKey: managedByValue,
					executionLabelKey: executionId,
				},
			},
			"spec": spec,
		},
	}
}

// findConflictingPolicy returns the name of an existing BackendTrafficPolicy that already targets the
// given HTTPRoute (and section), if any. Envoy Gateway resolves conflicts oldest-wins, so a pre-existing
// policy would silently shadow our attack — callers should fail when this returns a non-empty name.
// ownName is excluded from the check so re-running Start against our own just-created policy is a no-op.
func findConflictingPolicy(policies []unstructured.Unstructured, routeName, sectionName, ownName string) string {
	for i := range policies {
		policy := &policies[i]
		if policy.GetName() == ownName {
			continue
		}
		targetRefs, found, err := unstructured.NestedSlice(policy.Object, "spec", "targetRefs")
		if err != nil || !found {
			continue
		}
		for _, ref := range targetRefs {
			refMap, ok := ref.(map[string]any)
			if !ok {
				continue
			}
			if targetRefMatchesRoute(refMap, routeName, sectionName) {
				return policy.GetName()
			}
		}
	}
	return ""
}

func targetRefMatchesRoute(ref map[string]any, routeName, sectionName string) bool {
	kind, _ := ref["kind"].(string)
	name, _ := ref["name"].(string)
	if kind != httpRouteKind || name != routeName {
		return false
	}
	refSection, _ := ref["sectionName"].(string)
	// A whole-route policy conflicts with any section attack and vice versa; a section policy only
	// conflicts with the same section.
	if refSection == "" || sectionName == "" {
		return true
	}
	return refSection == sectionName
}

func objectMetaFromUnstructured(obj *unstructured.Unstructured) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Annotations: obj.GetAnnotations(),
		Labels:      obj.GetLabels(),
	}
}
