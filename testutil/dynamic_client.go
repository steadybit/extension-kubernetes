package testutil

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

// registerGVK registers a GroupVersionKind in the scheme
func registerGVK(scheme *runtime.Scheme, gvk schema.GroupVersionKind) {
	scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
}

// defaultListKinds maps the dynamic (CRD) resources the extension may watch to their list kinds.
// The fake dynamic client would otherwise naively pluralize kinds (e.g. Gateway -> "gatewaies"),
// which does not match the real resource names, so we register them explicitly. These are the
// resources whose informers are created by client.CreateClient when their (opt-in) features are
// enabled — in tests the feature flags default to their zero value (enabled), so the fake client
// must know these resources to avoid the informers panicking on LIST.
const gatewayNetworkingGroup = "gateway.networking.k8s.io"

var defaultListKinds = map[schema.GroupVersionResource]string{
	{Group: "argoproj.io", Version: "v1alpha1", Resource: "rollouts"}:                         "RolloutList",
	{Group: gatewayNetworkingGroup, Version: "v1", Resource: "httproutes"}:                    "HTTPRouteList",
	{Group: gatewayNetworkingGroup, Version: "v1", Resource: "gateways"}:                      "GatewayList",
	{Group: gatewayNetworkingGroup, Version: "v1", Resource: "gatewayclasses"}:                "GatewayClassList",
	{Group: "gateway.envoyproxy.io", Version: "v1alpha1", Resource: "backendtrafficpolicies"}: "BackendTrafficPolicyList",
}

// NewFakeDynamicClient creates a fake dynamic client. With no arguments it registers every CRD the
// extension may watch (Argo Rollouts and the Envoy Gateway resources) so that a client built from it
// does not panic when the corresponding informers LIST. To register only specific types instead,
// pass them as arguments:
//
//	NewFakeDynamicClient(
//		schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "CustomResource"},
//		schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "CustomResourceList"},
//	)
func NewFakeDynamicClient(gvks ...schema.GroupVersionKind) *fake.FakeDynamicClient {
	scheme := runtime.NewScheme()

	if len(gvks) == 0 {
		for gvr, listKind := range defaultListKinds {
			gv := gvr.GroupVersion()
			scheme.AddKnownTypeWithName(gv.WithKind(kindFromListKind(listKind)), &unstructured.Unstructured{})
			scheme.AddKnownTypeWithName(gv.WithKind(listKind), &unstructured.UnstructuredList{})
		}
		return fake.NewSimpleDynamicClientWithCustomListKinds(scheme, defaultListKinds)
	}

	// Register all provided GVKs
	for _, gvk := range gvks {
		registerGVK(scheme, gvk)
	}

	return fake.NewSimpleDynamicClient(scheme)
}

func kindFromListKind(listKind string) string {
	return listKind[:len(listKind)-len("List")]
}
