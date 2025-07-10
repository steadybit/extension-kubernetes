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

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
)

// TestNginxDelayTrafficAction_Prepare tests the Prepare method of NginxDelayTrafficAction
func TestNginxDelayTrafficAction_Prepare(t *testing.T) {
	// Setup test environment
	testEnv := setupNginxtestDelayEnvironment(t)
	defer testEnv.cleanup()

	// Define test cases
	tests := []struct {
		name        string
		ingressName string
		config      map[string]interface{}
		want        NginxDelayTrafficState
		wantErr     string
	}{
		{
			name:        "delay with path regex - open source nginx",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":        500,
				"conditionPathPattern": "/api/.*",
			},
			want: NginxDelayTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: myTestUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseDelay:        500,
				ConditionPathPattern: "/api/.*",
				IsEnterpriseNginx:    false,
			},
		},
		{
			name:        "delay with enterprise nginx",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":        500,
				"conditionPathPattern": "/api/.*",
				"isEnterpriseNginx":    true,
			},
			want: NginxDelayTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: myTestUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseDelay:        500,
				ConditionPathPattern: "/api/.*",
				IsEnterpriseNginx:    true,
			},
		},
		{
			name:        "delay with http method",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":       500,
				"conditionHttpMethod": "POST",
			},
			want: NginxDelayTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: myTestUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseDelay:       500,
				ConditionHttpMethod: "POST",
			},
		},
		{
			name:        "delay with http header",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay": 500,
				"conditionHttpHeader": []interface{}{
					map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
				},
			},
			want: NginxDelayTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: myTestUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseDelay: 500,
				ConditionHttpHeader: map[string]string{
					"User-Agent": "Mozilla.*",
				},
			},
		},
		{
			name:        "delay with multiple conditions",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":        1000,
				"conditionPathPattern": "/api/users",
				"conditionHttpMethod":  "POST",
				"conditionHttpHeader": []interface{}{
					map[string]interface{}{"key": "Content-Type", "value": "application/json"},
				},
			},
			want: NginxDelayTrafficState{
				NginxBaseState: NginxBaseState{
					ExecutionId: myTestUUID,
					Namespace:   "demo",
					IngressName: "test-nginx-ingress",
				},
				ResponseDelay:        1000,
				ConditionPathPattern: "/api/users",
				ConditionHttpMethod:  "POST",
				ConditionHttpHeader: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		{
			name:        "invalid delay value",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":        "abc",
				"conditionPathPattern": "/api/.*",
			},
			wantErr: "delay must be a number, got string: abc",
		},
		{
			name:        "no conditions provided",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay": 500,
			},
			wantErr: "at least one condition (path, method, or header) is required",
		},
		{
			name:        "path collision with existing rule",
			ingressName: "conflict-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":        500,
				"conditionPathPattern": "/alreadyDelayed",
			},
			wantErr: "a rule for path /alreadyDelayed already exists",
		},
		{
			name:        "duplicate delay rule",
			ingressName: "delay-conflict-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":        500,
				"conditionPathPattern": "/newPath",
			},
			wantErr: "a delay rule already exists - cannot add another one",
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test request
			request := createDelayNginxTestRequest(tt.ingressName, tt.config)

			// Run the Prepare method
			action := &NginxDelayTrafficAction{}
			state := action.NewEmptyState()
			_, err := action.Prepare(context.Background(), &state, request)

			// Verify results
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assertNginxDelayStateMatches(t, tt.want, state)
		})
	}
}

// TestNginxDelayTrafficAction_PrepareFluentChaining tests chained conditions
func TestNginxDelayTrafficAction_PrepareFluentChaining(t *testing.T) {
	// Setup test environment
	testEnv := setupNginxtestDelayEnvironment(t)
	defer testEnv.cleanup()

	// Test with all conditions to check the chained logic
	config := map[string]interface{}{
		"responseDelay":        500,
		"conditionPathPattern": "/api/users",
		"conditionHttpMethod":  "POST",
		"conditionHttpHeader": []interface{}{
			map[string]interface{}{"key": "Content-Type", "value": "application/json"},
		},
	}

	request := createDelayNginxTestRequest("test-nginx-ingress", config)

	// Run the Prepare method
	action := &NginxDelayTrafficAction{}
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Verify the configuration contains all conditions correctly chained
	assert.Contains(t, state.AnnotationConfig, "if ($request_uri ~* /api/users)")
	assert.Contains(t, state.AnnotationConfig, "if ($request_method != POST)")
	assert.Contains(t, state.AnnotationConfig, "if ($http_content_type !~* application/json)")

	// Check for unique variable names based on execution ID
	expectedSleepDurationVar := getNginxUniqueVariableName(state.ExecutionId, "sleep_ms_duration")
	assert.Contains(t, state.AnnotationConfig, fmt.Sprintf("sb_sleep_ms %s", expectedSleepDurationVar))
}

// Test Helpers

// testDelayEnvironment holds test resources
type testDelayEnvironment struct {
	stopCh  chan struct{}
	client  *client.Client
	cleanup func()
}

// myTestUUID is a fixed UUID for predictable test results
var myTestUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// setupNginxtestDelayEnvironment creates and configures the test environment for NGINX
func setupNginxtestDelayEnvironment(t *testing.T) *testDelayEnvironment {
	// Create test environment
	stopCh := make(chan struct{})
	testClient, clientset := getTestClient(stopCh)
	client.K8S = testClient

	// Create test ingresses
	createTestDelayNginxIngresses(t, clientset)

	// Wait for ingresses to be registered
	assert.Eventually(t, func() bool {
		ingress, _ := testClient.IngressByNamespaceAndName("demo", "test-nginx-ingress")
		return ingress != nil
	}, time.Second, 100*time.Millisecond)

	// Return environment with cleanup function
	return &testDelayEnvironment{
		stopCh: stopCh,
		client: testClient,
		cleanup: func() {
			close(stopCh)
		},
	}
}

// createTestDelayNginxIngresses creates test ingress resources for NGINX
func createTestDelayNginxIngresses(t *testing.T, clientset kubernetes.Interface) {
	// Regular ingress for most test cases
	createNginxIngress(t, clientset, "test-nginx-ingress", "# Some config\nif ($request_uri ~* /someOtherPath) {\n  return 404;\n}\n")

	// Ingress with existing path rule for testing conflicts
	createNginxIngress(t, clientset, "conflict-nginx-ingress", "location ~ /alreadyDelayed {\n  return 503;\n}\n")

	// Ingress with existing delay rule for testing conflicts
	createNginxIngress(t, clientset, "delay-conflict-nginx-ingress", "if ($request_uri ~* /existingPath) {\n  sb_sleep_ms 0.200;\n}\n")
}

// createDelayNginxTestRequest creates a test request with the given ingress name and config
func createDelayNginxTestRequest(ingressName string, config map[string]interface{}) action_kit_api.PrepareActionRequestBody {
	return action_kit_api.PrepareActionRequestBody{
		ExecutionId: myTestUUID,
		Config:      config,
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace": {"demo"},
				"k8s.ingress":   {ingressName},
			},
		}),
	}
}

// assertNginxDelayStateMatches verifies that the actual state matches the expected state
func assertNginxDelayStateMatches(t *testing.T, expected, actual NginxDelayTrafficState) {
	// Check basic properties
	assert.Equal(t, expected.ResponseDelay, actual.ResponseDelay)
	assert.Equal(t, expected.ConditionPathPattern, actual.ConditionPathPattern)
	assert.Equal(t, expected.ConditionHttpMethod, actual.ConditionHttpMethod)
	assert.Equal(t, expected.ConditionHttpHeader, actual.ConditionHttpHeader)
	assert.Equal(t, expected.Namespace, actual.Namespace)
	assert.Equal(t, expected.IngressName, actual.IngressName)
	assert.Equal(t, expected.IsEnterpriseNginx, actual.IsEnterpriseNginx)

	// Check annotation config contains expected elements
	assert.Contains(t, actual.AnnotationConfig, "# BEGIN STEADYBIT")
	assert.Contains(t, actual.AnnotationConfig, "# END STEADYBIT")
	assert.Contains(t, actual.AnnotationConfig, "sb_sleep_ms")

	// Generate expected unique variable names for this execution
	expectedShouldDelayVar := getNginxUniqueVariableName(actual.ExecutionId, "should_delay")
	expectedSleepDurationVar := getNginxUniqueVariableName(actual.ExecutionId, "sleep_ms_duration")

	assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("set %s", expectedShouldDelayVar))
	assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("set %s", expectedSleepDurationVar))
	assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("if (%s = 1)", expectedShouldDelayVar))
	assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("sb_sleep_ms %s", expectedSleepDurationVar))

	if actual.ConditionPathPattern != "" {
		assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("$request_uri ~* %s", actual.ConditionPathPattern))
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
