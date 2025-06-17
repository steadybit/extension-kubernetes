/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extingress

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TestNginxBlockTrafficAction_Prepare tests the Prepare method of NginxBlockTrafficAction
func TestNginxBlockTrafficAction_Prepare(t *testing.T) {
	// Setup test environment
	testEnv := setupNginxTestEnvironment(t)
	defer testEnv.cleanup()

	// Define test cases
	tests := []struct {
		name        string
		ingressName string
		config      map[string]interface{}
		want        NginxBlockTrafficState
		wantErr     string
	}{
		{
			name:        "block with path regex - open source nginx",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode":   503,
				"conditionPathPattern": "/api/.*",
			},
			want: NginxBlockTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseStatusCode:   503,
				ConditionPathPattern: "/api/.*",
				IsEnterpriseNginx:    false,
			},
		},
		{
			name:        "block with path regex - enterprise nginx",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode":   503,
				"conditionPathPattern": "/api/.*",
				"isEnterpriseNginx":    true,
			},
			want: NginxBlockTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseStatusCode:   503,
				ConditionPathPattern: "/api/.*",
				IsEnterpriseNginx:    true,
			},
		},
		{
			name:        "block with http method",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode":  503,
				"conditionHttpMethod": "POST",
			},
			want: NginxBlockTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseStatusCode:  503,
				ConditionHttpMethod: "POST",
			},
		},
		{
			name:        "block with http header",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode": 503,
				"conditionHttpHeader": []interface{}{
					map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
				},
			},
			want: NginxBlockTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseStatusCode: 503,
				ConditionHttpHeader: map[string]string{
					"User-Agent": "Mozilla.*",
				},
			},
		},
		{
			name:        "block with multiple conditions",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode":   503,
				"conditionPathPattern": "/api/users",
				"conditionHttpMethod":  "POST",
				"conditionHttpHeader": []interface{}{
					map[string]interface{}{"key": "Content-Type", "value": "application/json"},
				},
			},
			want: NginxBlockTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseStatusCode:   503,
				ConditionPathPattern: "/api/users",
				ConditionHttpMethod:  "POST",
				ConditionHttpHeader: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		{
			name:        "no conditions provided",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode": 503,
			},
			wantErr: "at least one condition (path, method, or header) is required",
		},
		{
			name:        "path collision with existing rule",
			ingressName: "conflict-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode":   503,
				"conditionPathPattern": "/alreadyBlocked",
			},
			wantErr: "a rule for path /alreadyBlocked already exists",
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test request
			request := createNginxTestRequest(tt.ingressName, tt.config)

			// Run the Prepare method
			action := &NginxBlockTrafficAction{}
			state := action.NewEmptyState()
			_, err := action.Prepare(context.Background(), &state, request)

			// Verify results
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assertNginxBlockStateMatches(t, tt.want, state)
		})
	}
}

// Fixed test UUID for predictable test results

// setupNginxTestEnvironment creates and configures the test environment for NGINX
func setupNginxTestEnvironment(t *testing.T) *testEnvironment {
	// Create test environment
	stopCh := make(chan struct{})
	testClient, clientset := getTestClient(stopCh)
	client.K8S = testClient

	// Create test ingresses
	createTestNginxIngresses(t, clientset)

	// Wait for ingresses to be registered
	assert.Eventually(t, func() bool {
		ingress, _ := testClient.IngressByNamespaceAndName("demo", "test-nginx-ingress")
		return ingress != nil
	}, time.Second, 100*time.Millisecond)

	// Return environment with cleanup function
	return &testEnvironment{
		stopCh: stopCh,
		client: testClient,
		cleanup: func() {
			close(stopCh)
		},
	}
}

// createTestNginxIngresses creates test ingress resources for NGINX
func createTestNginxIngresses(t *testing.T, clientset kubernetes.Interface) {
	// Regular ingress for most test cases
	createNginxIngress(t, clientset, "test-nginx-ingress", "# Some config\nif ($request_uri ~* /someOtherPath) {\n  return 404;\n}\n")

	// Ingress with existing path rule for testing conflicts
	createNginxIngress(t, clientset, "conflict-nginx-ingress", "location ~ /alreadyBlocked {\n  return 503;\n}\n")
}

// createNginxIngress creates a test ingress with the given name and config
func createNginxIngress(t *testing.T, clientset kubernetes.Interface, name, configSnippet string) {
	_, err := clientset.
		NetworkingV1().
		Ingresses("demo").
		Create(context.Background(), &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "demo",
				Annotations: map[string]string{
					"kubernetes.io/ingress.class": "nginx",
					NginxAnnotationKey:            configSnippet,
				},
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)
}

// createNginxTestRequest creates a test request with the given ingress name and config
func createNginxTestRequest(ingressName string, config map[string]interface{}) action_kit_api.PrepareActionRequestBody {
	return action_kit_api.PrepareActionRequestBody{
		ExecutionId: testUUID,
		Config:      config,
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace": {"demo"},
				"k8s.ingress":   {ingressName},
			},
		}),
	}
}

// assertNginxBlockStateMatches verifies that the actual state matches the expected state
func assertNginxBlockStateMatches(t *testing.T, expected, actual NginxBlockTrafficState) {
	// Check basic properties
	assert.Equal(t, expected.ResponseStatusCode, actual.ResponseStatusCode)
	assert.Equal(t, expected.ConditionPathPattern, actual.ConditionPathPattern)
	assert.Equal(t, expected.ConditionHttpMethod, actual.ConditionHttpMethod)
	assert.Equal(t, expected.ConditionHttpHeader, actual.ConditionHttpHeader)
	assert.Equal(t, expected.Namespace, actual.Namespace)
	assert.Equal(t, expected.IngressName, actual.IngressName)
	assert.Equal(t, expected.IsEnterpriseNginx, actual.IsEnterpriseNginx)

	// Check annotation config contains expected elements
	assert.Contains(t, actual.AnnotationConfig, "# BEGIN STEADYBIT")
	assert.Contains(t, actual.AnnotationConfig, "# END STEADYBIT")
	assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("return %d", actual.ResponseStatusCode))

	if actual.ConditionPathPattern != "" {
		if actual.IsEnterpriseNginx {
			assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("location ~ %s", actual.ConditionPathPattern))
		} else {
			assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("$request_uri ~* %s", actual.ConditionPathPattern))
		}
	}

	if actual.ConditionHttpMethod != "" {
		assert.Contains(t, actual.AnnotationConfig, "$request_method")
		assert.Contains(t, actual.AnnotationConfig, actual.ConditionHttpMethod)
	}

	for headerName, headerValue := range actual.ConditionHttpHeader {
		normalizedHeaderName := fmt.Sprintf("$http_%s", strings.Replace(strings.ToLower(headerName), "-", "_", -1))
		assert.Contains(t, actual.AnnotationConfig, normalizedHeaderName)
		assert.Contains(t, actual.AnnotationConfig, headerValue)
	}
}
