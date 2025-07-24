/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTransformIngressClass(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "transforms IngressClass with all relevant annotations",
			input: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx-steadybit",
					Annotations: map[string]string{
						"ingressclass.kubernetes.io/is-default-class": "true",
						"operator-sdk/primary-resource":              "nginx-system/nginx-controller",
						"meta.helm.sh/release-namespace":             "nginx-system",
						"meta.helm.sh/release-name":                  "nginx-ingress",
						"other.annotation/should-be-removed":         "some-value",
						"yet.another/annotation":                     "another-value",
					},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Manager: "kubectl",
						},
					},
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "nginx.org/ingress-controller",
				},
			},
			expected: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx-steadybit",
					Annotations: map[string]string{
						"ingressclass.kubernetes.io/is-default-class": "true",
						"operator-sdk/primary-resource":              "nginx-system/nginx-controller",
						"meta.helm.sh/release-namespace":             "nginx-system",
						"meta.helm.sh/release-name":                  "nginx-ingress",
					},
					ManagedFields: nil, // Should be cleared
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "nginx.org/ingress-controller",
				},
			},
			wantErr: false,
		},
		{
			name: "transforms IngressClass with subset of relevant annotations",
			input: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx-basic",
					Annotations: map[string]string{
						"ingressclass.kubernetes.io/is-default-class": "false",
						"meta.helm.sh/release-namespace":             "nginx-system",
						"irrelevant.annotation/should-be-removed":    "some-value",
					},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Manager: "helm",
						},
					},
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "k8s.io/ingress-nginx",
				},
			},
			expected: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx-basic",
					Annotations: map[string]string{
						"ingressclass.kubernetes.io/is-default-class": "false",
						"meta.helm.sh/release-namespace":             "nginx-system",
					},
					ManagedFields: nil,
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "k8s.io/ingress-nginx",
				},
			},
			wantErr: false,
		},
		{
			name: "transforms IngressClass with empty string annotations (should not be kept)",
			input: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx-empty-annotations",
					Annotations: map[string]string{
						"ingressclass.kubernetes.io/is-default-class": "",         // Empty - should not be kept
						"operator-sdk/primary-resource":              "namespace/controller", // Non-empty - should be kept
						"meta.helm.sh/release-namespace":             "",         // Empty - should not be kept
						"meta.helm.sh/release-name":                  "",         // Empty - should not be kept
						"other.annotation/irrelevant":                "value",    // Irrelevant - should not be kept
					},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Manager: "controller",
						},
					},
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "nginx.org/ingress-controller",
				},
			},
			expected: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx-empty-annotations",
					Annotations: map[string]string{
						"operator-sdk/primary-resource": "namespace/controller",
					},
					ManagedFields: nil,
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "nginx.org/ingress-controller",
				},
			},
			wantErr: false,
		},
		{
			name: "transforms IngressClass with no relevant annotations",
			input: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx-no-relevant-annotations",
					Annotations: map[string]string{
						"some.other.annotation/value":     "test",
						"another.irrelevant/annotation":   "value",
					},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Manager: "test",
						},
					},
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "k8s.io/ingress-nginx",
				},
			},
			expected: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:          "nginx-no-relevant-annotations",
					Annotations:   nil, // Should be nil when no relevant annotations
					ManagedFields: nil,
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "k8s.io/ingress-nginx",
				},
			},
			wantErr: false,
		},
		{
			name: "transforms IngressClass with nil annotations",
			input: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "nginx-nil-annotations",
					Annotations: nil,
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Manager: "test",
						},
					},
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "nginx.org/ingress-controller",
				},
			},
			expected: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:          "nginx-nil-annotations",
					Annotations:   nil,
					ManagedFields: nil,
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "nginx.org/ingress-controller",
				},
			},
			wantErr: false,
		},
		{
			name: "passes through non-IngressClass objects unchanged",
			input: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "some-ingress",
					Annotations: map[string]string{
						"should": "remain-unchanged",
					},
				},
			},
			expected: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "some-ingress",
					Annotations: map[string]string{
						"should": "remain-unchanged",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "passes through string objects unchanged",
			input:    "not-an-ingress-class",
			expected: "not-an-ingress-class",
			wantErr:  false,
		},
		{
			name:     "passes through nil unchanged",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformIngressClass(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test specific behavior details
func TestTransformIngressClass_SpecificBehaviors(t *testing.T) {
	t.Run("should preserve IngressClass spec unchanged", func(t *testing.T) {
		input := &networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test",
				Annotations: map[string]string{"irrelevant": "annotation"},
			},
			Spec: networkingv1.IngressClassSpec{
				Controller: "custom.controller/type",
				Parameters: &networkingv1.IngressClassParametersReference{
					APIGroup: stringPtr("example.com"),
					Kind:     "CustomParams",
					Name:     "my-params",
				},
			},
		}

		result, err := transformIngressClass(input)
		require.NoError(t, err)

		ic, ok := result.(*networkingv1.IngressClass)
		require.True(t, ok)
		
		// Spec should be preserved exactly
		assert.Equal(t, input.Spec, ic.Spec)
	})

	t.Run("should always clear ManagedFields", func(t *testing.T) {
		input := &networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				ManagedFields: []metav1.ManagedFieldsEntry{
					{Manager: "kubectl-client-side-apply"},
					{Manager: "helm"},
				},
			},
		}

		result, err := transformIngressClass(input)
		require.NoError(t, err)

		ic, ok := result.(*networkingv1.IngressClass)
		require.True(t, ok)
		
		assert.Nil(t, ic.ObjectMeta.ManagedFields)
	})

	t.Run("should preserve all annotation keys exactly", func(t *testing.T) {
		input := &networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Annotations: map[string]string{
					"ingressclass.kubernetes.io/is-default-class": "true",
					"operator-sdk/primary-resource":              "ns/deploy",
					"meta.helm.sh/release-namespace":             "helm-ns",
					"meta.helm.sh/release-name":                  "helm-release",
				},
			},
		}

		result, err := transformIngressClass(input)
		require.NoError(t, err)

		ic, ok := result.(*networkingv1.IngressClass)
		require.True(t, ok)
		
		expectedAnnotations := map[string]string{
			"ingressclass.kubernetes.io/is-default-class": "true",
			"operator-sdk/primary-resource":              "ns/deploy",
			"meta.helm.sh/release-namespace":             "helm-ns",
			"meta.helm.sh/release-name":                  "helm-release",
		}
		
		assert.Equal(t, expectedAnnotations, ic.Annotations)
	})
}

// Helper function for string pointer
func stringPtr(s string) *string {
	return &s
}