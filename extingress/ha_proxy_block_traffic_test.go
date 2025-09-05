// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestHAProxyBlockTrafficAction_Prepare tests the Prepare method of HAProxyBlockTrafficAction
func TestHAProxyBlockTrafficAction_Prepare(t *testing.T) {
	// Setup test environment
	testEnv := setupTestEnvironment(t)
	defer testEnv.cleanup()

	// Define test cases
	tests := []struct {
		name        string
		ingressName string
		config      map[string]interface{}
		want        HAProxyBlockTrafficState
		wantErr     string
	}{
		{
			name:        "block with path regex",
			ingressName: "test-ingress",
			config: map[string]interface{}{
				"responseStatusCode":   503,
				"conditionPathPattern": "/api/*",
			},
			want: HAProxyBlockTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-ingress",
				},
				ResponseStatusCode:   503,
				ConditionPathPattern: "/api/*",
				AnnotationConfig:     "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nacl sb_path_00000000-0000-0000-0000-000000000000 path_reg /api/*\nhttp-request return status 503 if sb_path_00000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
		},
		{
			name:        "block with http method",
			ingressName: "test-ingress",
			config: map[string]interface{}{
				"responseStatusCode":  503,
				"conditionHttpMethod": "POST",
			},
			want: HAProxyBlockTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-ingress",
				},
				ResponseStatusCode:  503,
				ConditionHttpMethod: "POST",
				AnnotationConfig:    "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nacl sb_method_00000000-0000-0000-0000-000000000000 method POST\nhttp-request return status 503 if sb_method_00000000-0000-0000-0000-000000000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
		},
		{
			name:        "block with http header",
			ingressName: "test-ingress",
			config: map[string]interface{}{
				"responseStatusCode": 503,
				"conditionHttpHeader": []interface{}{
					map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
				},
			},
			want: HAProxyBlockTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-ingress",
				},
				ResponseStatusCode: 503,
				ConditionHttpHeader: map[string]string{
					"User-Agent": "Mozilla.*",
				},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nacl sb_hdr_User_Agent_00000000-0000-0000-0000-000000000000 hdr(User-Agent) -m reg Mozilla.*\nhttp-request return status 503 if sb_hdr_User_Agent_00000000-0000-0000-0000-000000000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
		},
		{
			name:        "block with multiple conditions",
			ingressName: "test-ingress",
			config: map[string]interface{}{
				"responseStatusCode":   503,
				"conditionPathPattern": "/api/users",
				"conditionHttpMethod":  "POST",
				"conditionHttpHeader": []interface{}{
					map[string]interface{}{"key": "Content-Type", "value": "application/json"},
				},
			},
			want: HAProxyBlockTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: testUUID,
					Namespace:   "demo",
					IngressName: "test-ingress",
				},
				ResponseStatusCode:   503,
				ConditionPathPattern: "/api/users",
				ConditionHttpMethod:  "POST",
				ConditionHttpHeader: map[string]string{
					"Content-Type": "application/json",
				},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nacl sb_method_00000000-0000-0000-0000-000000000000 method POST\nacl sb_hdr_Content_Type_00000000-0000-0000-0000-000000000000 hdr(Content-Type) -m reg application/json\nacl sb_path_00000000-0000-0000-0000-000000000000 path_reg /api/users\nhttp-request return status 503 if sb_method_00000000-0000-0000-0000-000000000000 sb_hdr_Content_Type_00000000-0000-0000-0000-000000000000 sb_path_00000000-0000-0000-0000-000000000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
		},
		{
			name:        "no conditions provided",
			ingressName: "test-ingress",
			config: map[string]interface{}{
				"responseStatusCode": 503,
			},
			wantErr: "at least one condition (path, method, or header) is required",
		},
		{
			name:        "path collision with existing rule",
			ingressName: "conflict-ingress",
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
			request := createTestRequest(tt.ingressName, tt.config)

			// Run the Prepare method
			action := &HAProxyBlockTrafficAction{}
			state := action.NewEmptyState()
			_, err := action.Prepare(context.Background(), &state, request)

			// Verify results
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assertBlockStateMatches(t, tt.want, state)
		})
	}
}

// Fixed test UUID for predictable test results
var testUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// testEnvironment holds resources needed for testing
type testEnvironment struct {
	stopCh  chan struct{}
	client  *client.Client
	cleanup func()
}

// setupTestEnvironment creates and configures the test environment
func setupTestEnvironment(t *testing.T) *testEnvironment {
	// Create test environment
	stopCh := make(chan struct{})

	// Create test ingresses
	var objects []runtime.Object
	for _, obj := range createTestIngresses() {
		objects = append(objects, obj)
	}

	testClient := getTestClient(stopCh, objects...)
	client.K8S = testClient

	// Wait for ingresses to be registered
	assert.Eventually(t, func() bool {
		ingress, _ := testClient.IngressByNamespaceAndName("demo", "test-ingress")
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

// createTestIngresses creates test ingress resources
func createTestIngresses() []*networkingv1.Ingress {
	return []*networkingv1.Ingress{
		// Regular ingress for most test cases
		createIngress("test-ingress", "# Some config\nacl some_rule path_reg /someOtherPath\n"),

		// Ingress with existing path rule for testing conflicts
		createIngress("conflict-ingress", "acl sb_path_abcd path_reg /alreadyBlocked\nhttp-request return status 503 if { sb_path_abcd }\n"),
	}
}

// createIngress creates a test ingress with the given name and config
func createIngress(name, configSnippet string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "demo",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "haproxy",
				AnnotationKey:                 configSnippet,
			},
		},
	}
}

// createTestRequest creates a test request with the given ingress name and config
func createTestRequest(ingressName string, config map[string]interface{}) action_kit_api.PrepareActionRequestBody {
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

// assertBlockStateMatches verifies that the actual state matches the expected state
func assertBlockStateMatches(t *testing.T, expected, actual HAProxyBlockTrafficState) {
	// Check basic properties
	assert.Equal(t, expected.ResponseStatusCode, actual.ResponseStatusCode)
	assert.Equal(t, expected.ConditionPathPattern, actual.ConditionPathPattern)
	assert.Equal(t, expected.ConditionHttpMethod, actual.ConditionHttpMethod)
	assert.Equal(t, expected.ConditionHttpHeader, actual.ConditionHttpHeader)
	assert.Equal(t, expected.Namespace, actual.Namespace)
	assert.Equal(t, expected.IngressName, actual.IngressName)

	// Check annotation config contains expected elements
	if expected.AnnotationConfig != "" {
		assert.Contains(t, actual.AnnotationConfig, "# BEGIN STEADYBIT")
		assert.Contains(t, actual.AnnotationConfig, "# END STEADYBIT")
		assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("http-request return status %d", actual.ResponseStatusCode))

		if actual.ConditionPathPattern != "" {
			assert.Contains(t, actual.AnnotationConfig, "path_reg")
			assert.Contains(t, actual.AnnotationConfig, actual.ConditionPathPattern)
		}

		if actual.ConditionHttpMethod != "" {
			assert.Contains(t, actual.AnnotationConfig, "method")
			assert.Contains(t, actual.AnnotationConfig, actual.ConditionHttpMethod)
		}

		for headerName, headerValue := range actual.ConditionHttpHeader {
			assert.Contains(t, actual.AnnotationConfig, headerName)
			assert.Contains(t, actual.AnnotationConfig, headerValue)
		}
	}
}
