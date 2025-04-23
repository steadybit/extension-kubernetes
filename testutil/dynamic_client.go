package testutil

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

// NewFakeDynamicClient creates a fake dynamic client with additional custom types (i.e. Argo Rollouts)
func NewFakeDynamicClient() *fake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	gvk := schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Rollout",
	}
	scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	}, &unstructured.UnstructuredList{})
	return fake.NewSimpleDynamicClient(scheme)
}
