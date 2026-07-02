// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// EnvoyGatewayHttpRouteTargetType is the discovery target type for HTTPRoutes managed by Envoy Gateway.
	EnvoyGatewayHttpRouteTargetType = "com.steadybit.extension_kubernetes.kubernetes-envoy-gateway-httproute"

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

	// EnvoyGatewayIcon is a small inline gateway/route icon.
	EnvoyGatewayIcon = "data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%3E%3Cpath%20d%3D%22M4%206h4v4H4V6Zm12%208h4v4h-4v-4ZM6%2010v4h10v-4H6Z%22%20stroke%3D%22currentColor%22%20stroke-width%3D%221.5%22%2F%3E%3C%2Fsvg%3E"
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
