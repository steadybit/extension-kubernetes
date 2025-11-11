// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

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
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		want        NginxState
		wantErr     string
	}{
		{
			name:        "delay with path regex - open source nginx",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":        500,
				"conditionPathPattern": "/api/.*",
			},
			want: NginxState{
				ExecutionId:      myTestUUID,
				Namespace:        "demo",
				IngressName:      "test-nginx-ingress",
				Matcher:          RequestMatcher{PathPattern: "/api/.*"},
				AnnotationKey:    nginxAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\nset $sb_should_delay_00000000000000000000000000000000 1;\nif ($request_uri !~* /api/.*) { set $sb_should_delay_00000000000000000000000000000000 0; }\nset $sb_sleep_ms_duration_00000000000000000000000000000000 0;\nif ($sb_should_delay_00000000000000000000000000000000 = 1) { set $sb_sleep_ms_duration_00000000000000000000000000000000 500; }\nsb_sleep_ms $sb_sleep_ms_duration_00000000000000000000000000000000;\n# END STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\n",
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
			want: NginxState{
				ExecutionId:      myTestUUID,
				Namespace:        "demo",
				IngressName:      "test-nginx-ingress",
				Matcher:          RequestMatcher{PathPattern: "/api/.*"},
				AnnotationKey:    nginxEnterpriseAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\nset $sb_should_delay_00000000000000000000000000000000 1;\nif ($request_uri !~* /api/.*) { set $sb_should_delay_00000000000000000000000000000000 0; }\nset $sb_sleep_ms_duration_00000000000000000000000000000000 0;\nif ($sb_should_delay_00000000000000000000000000000000 = 1) { set $sb_sleep_ms_duration_00000000000000000000000000000000 500; }\nsb_sleep_ms $sb_sleep_ms_duration_00000000000000000000000000000000;\n# END STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\n",
			},
		},
		{
			name:        "delay with http method",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseDelay":       500,
				"conditionHttpMethod": "POST",
			},
			want: NginxState{
				ExecutionId:      myTestUUID,
				Namespace:        "demo",
				IngressName:      "test-nginx-ingress",
				Matcher:          RequestMatcher{HttpMethod: "POST"},
				AnnotationKey:    nginxAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\nset $sb_should_delay_00000000000000000000000000000000 1;\nif ($request_method != POST) { set $sb_should_delay_00000000000000000000000000000000 0; }\nset $sb_sleep_ms_duration_00000000000000000000000000000000 0;\nif ($sb_should_delay_00000000000000000000000000000000 = 1) { set $sb_sleep_ms_duration_00000000000000000000000000000000 500; }\nsb_sleep_ms $sb_sleep_ms_duration_00000000000000000000000000000000;\n# END STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\n",
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
			want: NginxState{
				ExecutionId: myTestUUID,
				Namespace:   "demo",
				IngressName: "test-nginx-ingress",
				Matcher: RequestMatcher{HttpHeader: map[string]string{
					"User-Agent": "Mozilla.*",
				},
				},
				AnnotationKey:    nginxAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\nset $sb_should_delay_00000000000000000000000000000000 1;\nif ($http_user_agent !~* Mozilla.*) { set $sb_should_delay_00000000000000000000000000000000 0; }\nset $sb_sleep_ms_duration_00000000000000000000000000000000 0;\nif ($sb_should_delay_00000000000000000000000000000000 = 1) { set $sb_sleep_ms_duration_00000000000000000000000000000000 500; }\nsb_sleep_ms $sb_sleep_ms_duration_00000000000000000000000000000000;\n# END STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\n",
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
			want: NginxState{
				ExecutionId: myTestUUID,
				Namespace:   "demo",
				IngressName: "test-nginx-ingress",
				Matcher: RequestMatcher{PathPattern: "/api/users",
					HttpMethod: "POST",
					HttpHeader: map[string]string{
						"Content-Type": "application/json",
					},
				},
				AnnotationKey:    nginxAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\nset $sb_should_delay_00000000000000000000000000000000 1;\nif ($request_uri !~* /api/users) { set $sb_should_delay_00000000000000000000000000000000 0; }\nif ($request_method != POST) { set $sb_should_delay_00000000000000000000000000000000 0; }\nif ($http_content_type !~* application/json) { set $sb_should_delay_00000000000000000000000000000000 0; }\nset $sb_sleep_ms_duration_00000000000000000000000000000000 0;\nif ($sb_should_delay_00000000000000000000000000000000 = 1) { set $sb_sleep_ms_duration_00000000000000000000000000000000 1000; }\nsb_sleep_ms $sb_sleep_ms_duration_00000000000000000000000000000000;\n# END STEADYBIT - Delay - 00000000-0000-0000-0000-000000000000\n",
			},
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
			action := NewNginxDelayTrafficAction()
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
	action := NewNginxDelayTrafficAction()
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Verify the configuration contains all conditions correctly chained
	assert.Contains(t, state.AnnotationConfig, "if ($request_uri !~* /api/users)")
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

	var objects []runtime.Object
	// Create test IngressClass for NGINX
	objects = append(objects, createTestNginxIngressClass())

	// Create test ingresses
	for _, ingress := range createTestDelayNginxIngresses() {
		objects = append(objects, ingress)
	}

	testClient := getTestClient(stopCh, objects...)
	client.K8S = testClient

	// Use no-op validator for tests
	extconfig.Config.NginxDelaySkipImageCheck = true

	// Wait for IngressClass and ingresses to be registered
	assert.Eventually(t, func() bool {
		ingressClasses := testClient.IngressClasses()
		return len(ingressClasses) > 0
	}, time.Second, 100*time.Millisecond)

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
			extconfig.Config.NginxDelaySkipImageCheck = false
		},
	}
}

// createTestNginxIngressClass creates a test NGINX IngressClass
func createTestNginxIngressClass() *networkingv1.IngressClass {
	return &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
		},
		Spec: networkingv1.IngressClassSpec{
			Controller: "k8s.io/ingress-nginx",
		},
	}
}

// createTestDelayNginxIngresses creates test ingress resources for NGINX
func createTestDelayNginxIngresses() []*networkingv1.Ingress {
	return []*networkingv1.Ingress{
		// Regular ingress for most test cases
		createNginxIngress("test-nginx-ingress", "# Some config\nif ($request_uri ~* /someOtherPath) {\n  return 404;\n}\n"),

		// Ingress with existing path rule for testing conflicts
		createNginxIngress("conflict-nginx-ingress", "location ~ /alreadyDelayed {\n  return 503;\n}\n"),

		// Ingress with existing delay rule for testing conflicts
		createNginxIngress("delay-conflict-nginx-ingress", "if ($request_uri ~* /existingPath) {\n  sb_sleep_ms 0.200;\n}\n"),
	}
}

// createDelayNginxTestRequest creates a test request with the given ingress name and config
func createDelayNginxTestRequest(ingressName string, config map[string]interface{}) action_kit_api.PrepareActionRequestBody {
	return action_kit_api.PrepareActionRequestBody{
		ExecutionId: myTestUUID,
		Config:      config,
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace":     {"demo"},
				"k8s.ingress":       {ingressName},
				"k8s.ingress.class": {"nginx"},
			},
		}),
	}
}

// assertNginxDelayStateMatches verifies that the actual state matches the expected state
func assertNginxDelayStateMatches(t *testing.T, expected, actual NginxState) {
	// Check basic properties
	assert.Equal(t, expected.Matcher.PathPattern, actual.Matcher.PathPattern)
	assert.Equal(t, expected.Matcher.HttpMethod, actual.Matcher.HttpMethod)
	assert.Equal(t, expected.Matcher.HttpHeader, actual.Matcher.HttpHeader)
	assert.Equal(t, expected.Namespace, actual.Namespace)
	assert.Equal(t, expected.IngressName, actual.IngressName)
	assert.Equal(t, expected.AnnotationKey, actual.AnnotationKey)
	assert.Equal(t, expected.AnnotationConfig, actual.AnnotationConfig)

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

	if actual.Matcher.PathPattern != "" {
		assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("$request_uri !~* %s", actual.Matcher.PathPattern))
	}

	if actual.Matcher.HttpMethod != "" {
		assert.Contains(t, actual.AnnotationConfig, "$request_method")
		assert.Contains(t, actual.AnnotationConfig, actual.Matcher.HttpMethod)
	}

	for headerName, headerValue := range actual.Matcher.HttpHeader {
		normalizedHeaderName := fmt.Sprintf("$http_%s", strings.Replace(strings.ToLower(headerName), "-", "_", -1))
		assert.Contains(t, actual.AnnotationConfig, normalizedHeaderName)
		assert.Contains(t, actual.AnnotationConfig, headerValue)
	}
}
