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
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		want        NginxState
		wantErr     string
	}{
		{
			name:        "block with path regex - open source nginx",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode":   503,
				"conditionPathPattern": "/api/.*",
			},
			want: NginxState{
				ExecutionId:      testUUIDBlock,
				Namespace:        "demo",
				IngressName:      "test-nginx-ingress",
				Matcher:          RequestMatcher{PathPattern: "/api/.*"},
				AnnotationKey:    nginxAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Block - 00000000-0000-0000-0000-000000000000\nset $sb_should_block_00000000000000000000000000000000 1;\nif ($request_uri !~* /api/.*) { set $sb_should_block_00000000000000000000000000000000 0; }\nif ($sb_should_block_00000000000000000000000000000000 = 1) { return 503; }\n# END STEADYBIT - Block - 00000000-0000-0000-0000-000000000000\n",
			},
		},
		{
			name:        "block with http method",
			ingressName: "test-nginx-ingress",
			config: map[string]interface{}{
				"responseStatusCode":  503,
				"conditionHttpMethod": "POST",
			},
			want: NginxState{
				ExecutionId:      testUUIDBlock,
				Namespace:        "demo",
				IngressName:      "test-nginx-ingress",
				Matcher:          RequestMatcher{HttpMethod: "POST"},
				AnnotationKey:    nginxAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Block - 00000000-0000-0000-0000-000000000000\nset $sb_should_block_00000000000000000000000000000000 1;\nif ($request_method != POST) { set $sb_should_block_00000000000000000000000000000000 0; }\nif ($sb_should_block_00000000000000000000000000000000 = 1) { return 503; }\n# END STEADYBIT - Block - 00000000-0000-0000-0000-000000000000\n",
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
			want: NginxState{
				ExecutionId: testUUIDBlock,
				Namespace:   "demo",
				IngressName: "test-nginx-ingress",
				Matcher: RequestMatcher{HttpHeader: map[string]string{
					"User-Agent": "Mozilla.*",
				}},
				AnnotationKey:    nginxAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Block - 00000000-0000-0000-0000-000000000000\nset $sb_should_block_00000000000000000000000000000000 1;\nif ($http_user_agent !~* Mozilla.*) { set $sb_should_block_00000000000000000000000000000000 0; }\nif ($sb_should_block_00000000000000000000000000000000 = 1) { return 503; }\n# END STEADYBIT - Block - 00000000-0000-0000-0000-000000000000\n",
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
			want: NginxState{
				ExecutionId: testUUIDBlock,
				Namespace:   "demo",
				IngressName: "test-nginx-ingress",
				Matcher: RequestMatcher{PathPattern: "/api/users",
					HttpMethod: "POST",
					HttpHeader: map[string]string{
						"Content-Type": "application/json",
					},
				},
				AnnotationKey:    nginxAnnotationKey,
				AnnotationConfig: "# BEGIN STEADYBIT - Block - 00000000-0000-0000-0000-000000000000\nset $sb_should_block_00000000000000000000000000000000 1;\nif ($request_uri !~* /api/users) { set $sb_should_block_00000000000000000000000000000000 0; }\nif ($request_method != POST) { set $sb_should_block_00000000000000000000000000000000 0; }\nif ($http_content_type !~* application/json) { set $sb_should_block_00000000000000000000000000000000 0; }\nif ($sb_should_block_00000000000000000000000000000000 = 1) { return 503; }\n# END STEADYBIT - Block - 00000000-0000-0000-0000-000000000000\n",
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
			action := NewNginxBlockTrafficAction()
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
var testUUIDBlock = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// setupNginxTestEnvironment creates and configures the test environment for NGINX
func setupNginxTestEnvironment(t *testing.T) *testEnvironment {
	// Create test environment
	stopCh := make(chan struct{})

	// Create test ingresses
	var objects []runtime.Object
	for _, obj := range createTestNginxIngresses() {
		objects = append(objects, obj)
	}

	testClient := getTestClient(stopCh, objects...)
	client.K8S = testClient

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
func createTestNginxIngresses() []*networkingv1.Ingress {
	return []*networkingv1.Ingress{
		// Regular ingress for most test cases
		createNginxIngress("test-nginx-ingress", "# Some config\nif ($request_uri ~* /someOtherPath) {\n  return 404;\n}\n"),

		// Ingress with existing path rule for testing conflicts
		createNginxIngress("conflict-nginx-ingress", "location ~ /alreadyBlocked {\n  return 503;\n}\n"),
	}
}

// createNginxIngress creates a test ingress with the given name and config
func createNginxIngress(name, configSnippet string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "demo",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
				nginxAnnotationKey:            configSnippet,
			},
		},
	}
}

// createNginxTestRequest creates a test request with the given ingress name and config
func createNginxTestRequest(ingressName string, config map[string]interface{}) action_kit_api.PrepareActionRequestBody {
	return action_kit_api.PrepareActionRequestBody{
		ExecutionId: testUUIDBlock,
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
func assertNginxBlockStateMatches(t *testing.T, expected, actual NginxState) {
	// Check basic properties
	assert.Equal(t, expected.Matcher.PathPattern, actual.Matcher.PathPattern)
	assert.Equal(t, expected.Matcher.HttpMethod, actual.Matcher.HttpMethod)
	assert.Equal(t, expected.Matcher.HttpHeader, actual.Matcher.HttpHeader)
	assert.Equal(t, expected.Namespace, actual.Namespace)
	assert.Equal(t, expected.IngressName, actual.IngressName)
	assert.Equal(t, expected.AnnotationKey, actual.AnnotationKey)
	assert.Equal(t, expected.AnnotationConfig, actual.AnnotationConfig)

	// Generate expected unique variable name for this execution
	expectedShouldBlockVar := getNginxUniqueVariableName(actual.ExecutionId, "should_block")
	assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("set %s", expectedShouldBlockVar))
	assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("if (%s = 1)", expectedShouldBlockVar))

	if actual.Matcher.PathPattern != "" {
		if actual.AnnotationKey == nginxEnterpriseAnnotationKey {
			assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("location ~ %s", actual.Matcher.PathPattern))
		} else {
			assert.Contains(t, actual.AnnotationConfig, fmt.Sprintf("$request_uri !~* %s", actual.Matcher.PathPattern))
		}
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
