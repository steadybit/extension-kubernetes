/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package client

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

func TestRemoveAnnotationBlock(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		startMarker string
		endMarker   string
		expected    string
	}{
		{
			name: "basic removal",
			config: `prefix text
# BEGIN STEADYBIT - abc123
some config to remove
more config
# END STEADYBIT - abc123
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "prefix text\nsuffix text",
		},
		{
			name: "markers not found",
			config: `prefix text
# BEGIN STEADYBIT - xyz789
some other config
# END STEADYBIT - xyz789
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected: `prefix text
# BEGIN STEADYBIT - xyz789
some other config
# END STEADYBIT - xyz789
suffix text`,
		},
		{
			name: "only start marker",
			config: `prefix text
# BEGIN STEADYBIT - abc123
some config
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected: `prefix text
# BEGIN STEADYBIT - abc123
some config
suffix text`,
		},
		{
			name: "only end marker",
			config: `prefix text
some config
# END STEADYBIT - abc123
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected: `prefix text
some config
# END STEADYBIT - abc123
suffix text`,
		},
		{
			name: "with trailing newlines",
			config: `prefix text
# BEGIN STEADYBIT - abc123
some config
# END STEADYBIT - abc123


suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "prefix text\nsuffix text",
		},
		{
			name: "multiple blocks",
			config: `prefix text
# BEGIN STEADYBIT - abc123
first block
# END STEADYBIT - abc123
middle text
# BEGIN STEADYBIT - abc123
second block
# END STEADYBIT - abc123
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected: `prefix text
middle text
# BEGIN STEADYBIT - abc123
second block
# END STEADYBIT - abc123
suffix text`,
		},
		{
			name:        "empty config",
			config:      "",
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "",
		},
		{
			name: "block at start",
			config: `# BEGIN STEADYBIT - abc123
some config
# END STEADYBIT - abc123
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "suffix text",
		},
		{
			name: "block at end",
			config: `prefix text
# BEGIN STEADYBIT - abc123
some config
# END STEADYBIT - abc123`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "prefix text\n",
		},
		{
			name: "only the block",
			config: `# BEGIN STEADYBIT - abc123
some config
# END STEADYBIT - abc123`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeAnnotationBlock(tt.config, tt.startMarker, tt.endMarker)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveAnnotationBlockMethod(t *testing.T) {
	// Create a test executionId
	executionId := uuid.MustParse("12345678-1234-1234-1234-123456789012")

	// Define test cases
	tests := []struct {
		name           string
		ingressName    string
		namespace      string
		annotations    map[string]string
		annotationKey  string
		setupReactor   func(*fake.Clientset)
		expectErr      bool
		expectErrMsg   string
		expectedConfig string
	}{
		{
			name:          "successfully removes annotation block",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "prefix\n# BEGIN STEADYBIT - 12345678-1234-1234-1234-123456789012\nsome config\n# END STEADYBIT - 12345678-1234-1234-1234-123456789012\nsuffix",
			},
			expectErr:      false,
			expectedConfig: "prefix\nsuffix",
		},
		{
			name:          "ingress not found",
			ingressName:   "nonexistent-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations:   map[string]string{},
			expectErr:     true,
			expectErrMsg:  "failed to fetch ingress: ingress default/nonexistent-ingress not found",
		},
		{
			name:          "annotation doesn't exist",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "nonexistent.annotation",
			annotations: map[string]string{
				"haproxy.org/config": "some config",
			},
			expectErr:      false,
			expectedConfig: "", // No change expected
		},
		{
			name:          "markers not found in annotation",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "some config without markers",
			},
			expectErr:      false,
			expectedConfig: "some config without markers", // No change expected
		},
		{
			name:          "handles conflict error and retries successfully",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "prefix\n# BEGIN STEADYBIT - 12345678-1234-1234-1234-123456789012\nsome config\n# END STEADYBIT - 12345678-1234-1234-1234-123456789012\nsuffix",
			},
			setupReactor: func(clientset *fake.Clientset) {
				conflictCount := 0
				clientset.PrependReactor("update", "ingresses", func(action k8sTesting.Action) (bool, runtime.Object, error) {
					if conflictCount < 1 {
						conflictCount++
						return true, nil, k8sErrors.NewConflict(
							schema.GroupResource{Group: "networking.k8s.io", Resource: "ingresses"},
							"test-ingress",
							errors.New("conflict error"),
						)
					}
					return false, nil, nil
				})
			},
			expectErr:      false,
			expectedConfig: "prefix\nsuffix",
		},
		{
			name:          "update fails with non-conflict error",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "prefix\n# BEGIN STEADYBIT - 12345678-1234-1234-1234-123456789012\nsome config\n# END STEADYBIT - 12345678-1234-1234-1234-123456789012\nsuffix",
			},
			setupReactor: func(clientset *fake.Clientset) {
				clientset.PrependReactor("update", "ingresses", func(action k8sTesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("permission denied")
				})
			},
			expectErr:    true,
			expectErrMsg: "failed to update ingress annotation: permission denied",
		},
		{
			name:          "too many conflicts",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "prefix\n# BEGIN STEADYBIT - 12345678-1234-1234-1234-123456789012\nsome config\n# END STEADYBIT - 12345678-1234-1234-1234-123456789012\nsuffix",
			},
			setupReactor: func(clientset *fake.Clientset) {
				clientset.PrependReactor("update", "ingresses", func(action k8sTesting.Action) (bool, runtime.Object, error) {
					return true, nil, k8sErrors.NewConflict(
						schema.GroupResource{Group: "networking.k8s.io", Resource: "ingresses"},
						"test-ingress",
						errors.New("conflict error"),
					)
				})
			},
			expectErr:    true,
			expectErrMsg: "failed to update ingress annotation after 10 attempts due to concurrent modifications",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake clientset
			clientset := fake.NewClientset()

			// Create the client
			client := &Client{
				networkingV1: clientset.NetworkingV1(),
			}

			// Setup any custom reactors for this test
			if tt.setupReactor != nil {
				tt.setupReactor(clientset)
			}

			// Create the ingress if the name is not "nonexistent-ingress"
			if tt.ingressName != "nonexistent-ingress" {
				_, err := clientset.NetworkingV1().Ingresses(tt.namespace).Create(
					context.Background(),
					&networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:        tt.ingressName,
							Namespace:   tt.namespace,
							Annotations: tt.annotations,
						},
					},
					metav1.CreateOptions{},
				)
				require.NoError(t, err)
			}

			// Call the method being tested
			err := client.RemoveIngressAnnotationBlock(
				context.Background(),
				tt.namespace,
				tt.ingressName,
				tt.annotationKey,
				executionId,
				fmt.Sprintf("# BEGIN STEADYBIT - %s", executionId.String()),
				fmt.Sprintf("# END STEADYBIT - %s", executionId.String()),
			)

			// Verify the result
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErrMsg)
			} else {
				require.NoError(t, err)

				// Only verify the updated ingress if we expect it to exist
				if tt.ingressName != "nonexistent-ingress" {
					// Get the updated ingress
					updatedIngress, err := clientset.NetworkingV1().Ingresses(tt.namespace).Get(
						context.Background(),
						tt.ingressName,
						metav1.GetOptions{},
					)
					require.NoError(t, err)

					// Verify annotation was updated correctly
					if tt.annotationKey != "nonexistent.annotation" {
						actualConfig := updatedIngress.Annotations[tt.annotationKey]
						assert.Equal(t, tt.expectedConfig, actualConfig)
					}
				}
			}
		})
	}
}

func TestUpdateIngressAnnotation(t *testing.T) {
	tests := []struct {
		name           string
		ingressName    string
		namespace      string
		annotations    map[string]string
		annotationKey  string
		newAnnotation  string
		setupReactor   func(*fake.Clientset)
		expectErr      bool
		expectErrMsg   string
		expectedConfig string
	}{
		{
			name:          "successfully updates existing annotation",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "existing config",
			},
			newAnnotation:  "prefix config",
			expectErr:      false,
			expectedConfig: "prefix config\nexisting config",
		},
		{
			name:           "successfully adds annotation when none exists",
			ingressName:    "test-ingress",
			namespace:      "default",
			annotationKey:  "haproxy.org/config",
			annotations:    map[string]string{},
			newAnnotation:  "new config",
			expectErr:      false,
			expectedConfig: "new config",
		},
		{
			name:          "ingress not found",
			ingressName:   "nonexistent-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations:   map[string]string{},
			newAnnotation: "new config",
			expectErr:     true,
			expectErrMsg:  "failed to fetch ingress: ingress default/nonexistent-ingress not found",
		},
		{
			name:          "handles conflict error and retries successfully",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "existing config",
			},
			newAnnotation: "prefix config",
			setupReactor: func(clientset *fake.Clientset) {
				conflictCount := 0
				clientset.PrependReactor("update", "ingresses", func(action k8sTesting.Action) (bool, runtime.Object, error) {
					if conflictCount < 1 {
						conflictCount++
						return true, nil, k8sErrors.NewConflict(
							schema.GroupResource{Group: "networking.k8s.io", Resource: "ingresses"},
							"test-ingress",
							errors.New("conflict error"),
						)
					}
					return false, nil, nil
				})
			},
			expectErr:      false,
			expectedConfig: "prefix config\nexisting config",
		},
		{
			name:          "update fails with non-conflict error",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "existing config",
			},
			newAnnotation: "prefix config",
			setupReactor: func(clientset *fake.Clientset) {
				clientset.PrependReactor("update", "ingresses", func(action k8sTesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("permission denied")
				})
			},
			expectErr:    true,
			expectErrMsg: "failed to update ingress annotation: permission denied",
		},
		{
			name:          "too many conflicts",
			ingressName:   "test-ingress",
			namespace:     "default",
			annotationKey: "haproxy.org/config",
			annotations: map[string]string{
				"haproxy.org/config": "existing config",
			},
			newAnnotation: "prefix config",
			setupReactor: func(clientset *fake.Clientset) {
				clientset.PrependReactor("update", "ingresses", func(action k8sTesting.Action) (bool, runtime.Object, error) {
					return true, nil, k8sErrors.NewConflict(
						schema.GroupResource{Group: "networking.k8s.io", Resource: "ingresses"},
						"test-ingress",
						errors.New("conflict error"),
					)
				})
			},
			expectErr:    true,
			expectErrMsg: "failed to update ingress annotation after 10 attempts due to concurrent modifications",
		},
		{
			name:           "handles nil annotations",
			ingressName:    "test-ingress-nil-annotations",
			namespace:      "default",
			annotationKey:  "haproxy.org/config",
			annotations:    nil,
			newAnnotation:  "new config",
			expectErr:      false,
			expectedConfig: "new config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake clientset
			clientset := fake.NewSimpleClientset()

			// Create the client
			client := &Client{
				networkingV1: clientset.NetworkingV1(),
			}

			// Setup any custom reactors for this test
			if tt.setupReactor != nil {
				tt.setupReactor(clientset)
			}

			// Create the ingress if the name is not "nonexistent-ingress"
			if tt.ingressName != "nonexistent-ingress" {
				_, err := clientset.NetworkingV1().Ingresses(tt.namespace).Create(
					context.Background(),
					&networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:        tt.ingressName,
							Namespace:   tt.namespace,
							Annotations: tt.annotations,
						},
					},
					metav1.CreateOptions{},
				)
				require.NoError(t, err)
			}

			// Call the method being tested
			_, err := client.UpdateIngressAnnotation(
				context.Background(),
				tt.namespace,
				tt.ingressName,
				tt.annotationKey,
				tt.newAnnotation,
			)

			// Verify the result
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErrMsg)
			} else {
				require.NoError(t, err)

				// Only verify the updated ingress if we expect it to exist
				if tt.ingressName != "nonexistent-ingress" {
					// Get the updated ingress
					updatedIngress, err := clientset.NetworkingV1().Ingresses(tt.namespace).Get(
						context.Background(),
						tt.ingressName,
						metav1.GetOptions{},
					)
					require.NoError(t, err)

					// Verify annotation was updated correctly
					assert.Equal(t, tt.expectedConfig, updatedIngress.Annotations[tt.annotationKey])
				}
			}
		})
	}
}
