package testutil

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

// registerGVK registers a GroupVersionKind  in the scheme
func registerGVK(scheme *runtime.Scheme, gvk schema.GroupVersionKind) {
	scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
}

// NewFakeDynamicClient creates a fake dynamic client with additional custom types.
// If no GVKs are provided, it defaults to Argo Rollouts for backward compatibility.
// To add additional types, pass them as arguments:
//
//	NewFakeDynamicClient(
//		schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Rollout"},
//		schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "CustomResource"},
//	)
func NewFakeDynamicClient(gvks ...schema.GroupVersionKind) *fake.FakeDynamicClient {
	scheme := runtime.NewScheme()

	// Default to Argo Rollout  if no GVKs provided
	if len(gvks) == 0 {
		gvks = []schema.GroupVersionKind{
			{
				Group:   "argoproj.io",
				Version: "v1alpha1",
				Kind:    "Rollout",
			},
			{
				Group:   "argoproj.io",
				Version: "v1alpha1",
				Kind:    "RolloutList",
			},
		}
	}

	// Register all provided GVKs
	for _, gvk := range gvks {
		registerGVK(scheme, gvk)
	}

	return fake.NewSimpleDynamicClient(scheme)
}
