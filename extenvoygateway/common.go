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

	DelayActionId = "com.steadybit.extension_kubernetes.envoy-gateway-http-route-delay"
	AbortActionId = "com.steadybit.extension_kubernetes.envoy-gateway-http-route-abort"

	// envoyGatewayControllerName identifies GatewayClasses managed by Envoy Gateway.
	envoyGatewayControllerName = "gateway.envoyproxy.io/gatewayclass-controller"

	gatewayAPIGroup   = "gateway.networking.k8s.io"
	httpRouteKind     = "HTTPRoute"
	btpAPIVersion     = "gateway.envoyproxy.io/v1alpha1"
	btpKind           = "BackendTrafficPolicy"
	managedByLabelKey = "steadybit.com/managed-by"
	managedByValue    = "extension-kubernetes"
	executionLabelKey = "steadybit.com/execution-id"

	// EnvoyGatewayIcon is the Envoy Gateway logo (monochrome, currentColor).
	EnvoyGatewayIcon = "data:image/svg+xml,%3Csvg%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M4.20117%2012.9791L2.70606%2013.6373L2.78125%2016.8659L5.8252%2018.7926L7.1709%2018.1989L8.38672%2018.9694L6.14746%2019.9528C6.13256%2019.9629%206.11752%2019.9682%206.09766%2019.9733C5.9289%2020.0289%205.7205%2019.9986%205.55176%2019.8922L1.90234%2017.5807C1.7137%2017.4591%201.59389%2017.2617%201.58887%2017.0641L1.5%2013.1862C1.49518%2012.9887%201.60447%2012.816%201.78809%2012.735L4.1709%2011.6862L4.20117%2012.9791Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3Cpath%20d%3D%22M8.01465%2010.2164L6.1377%2011.0426L6.23731%2015.4166L10.3535%2018.0221L12.1611%2017.2262L13.5811%2018.1237L10.7305%2019.3756C10.7107%2019.3857%2010.6906%2019.395%2010.6709%2019.4C10.4673%2019.4608%2010.2289%2019.431%2010.0303%2019.3043L5.2041%2016.2477C4.98572%2016.1109%204.84685%2015.8831%204.8418%2015.65L4.72754%2010.5202C4.72271%2010.2872%204.85202%2010.09%205.06543%209.99378L7.97949%208.71058L8.01465%2010.2164Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3Cpath%20d%3D%22M15.2988%2011.0475C15.5173%2011.1894%2015.6562%2011.4179%2015.6611%2011.651L15.7656%2016.3287L15.6914%2016.359L14.3506%2015.5133L14.2666%2011.8844L10.7559%209.66371L10.7256%208.30531L10.8652%208.24476L15.2988%2011.0475Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M15.0107%204.03773C15.2441%203.9617%2015.5177%204.00226%2015.7461%204.14418L21.9326%208.06214C22.1859%208.21927%2022.3456%208.48334%2022.3506%208.74183L22.499%2015.3102C22.504%2015.5737%2022.3555%2015.8072%2022.1123%2015.9137L16.0732%2018.569C16.0486%2018.5791%2016.0286%2018.5893%2016.0039%2018.5944C15.7755%2018.6703%2015.5028%2018.6298%2015.2744%2018.4879L9.08691%2014.5748C8.83394%2014.4177%208.67495%2014.1597%208.66992%2013.8912L8.51563%207.32191C8.51082%207.05351%208.66022%206.82578%208.90332%206.71937L14.9414%204.06312C14.9661%204.05302%2014.986%204.04282%2015.0107%204.03773ZM10.1348%207.91566L10.2686%2013.6276L15.6465%2017.0289L20.8906%2014.7223L20.7559%209.01039L15.3789%205.60902L10.1348%207.91566Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3Cpath%20d%3D%22M8.78418%2015.0875L9.95117%2015.8268L9.98047%2017.0895L8.76465%2016.319L8.73438%2015.0514C8.74927%2015.0615%208.76928%2015.0774%208.78418%2015.0875Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3Cpath%20d%3D%22M8.06445%2012.568L8.09375%2013.8551L6.76367%2013.0143L6.7334%2011.7272L8.06445%2012.568Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3C%2Fsvg%3E"
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
