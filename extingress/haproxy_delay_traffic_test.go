// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"context"
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
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestHAProxyDelayTrafficAction_Prepare(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	objects := []runtime.Object{
		// Create a simple ingress without delay rules for regular test cases
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "simple-ingress",
				Namespace: "demo",
				Annotations: map[string]string{
					"kubernetes.io/ingress.class": "haproxy",
					haProxyAnnotationKey:          "# Some other config\nacl some_rule path_reg /anotherPath\n",
				},
			},
		},
		// Create an ingress with a path conflict for path collision test
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "path-conflict-ingress",
				Namespace: "demo",
				Annotations: map[string]string{
					"kubernetes.io/ingress.class": "haproxy",
					haProxyAnnotationKey:          "acl path_conflict path_reg /alreadyDelayed\n",
				},
			},
		},
		// Create an ingress with an existing delay rule for duplicate delay test
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "delay-conflict-ingress",
				Namespace: "demo",
				Annotations: map[string]string{
					"kubernetes.io/ingress.class": "haproxy",
					haProxyAnnotationKey:          "tcp-request inspect-delay 1000ms\ntcp-request content accept if WAIT_END\n",
				},
			},
		},
	}

	testClient := getTestClient(stopCh, objects...)
	client.K8S = testClient

	assert.Eventually(t, func() bool {
		ingress, _ := testClient.IngressByNamespaceAndName("demo", "simple-ingress")
		return ingress != nil
	}, time.Second, 100*time.Millisecond)

	type args struct {
		in0     context.Context
		state   *HAProxyState
		request action_kit_api.PrepareActionRequestBody
	}
	tests := []struct {
		name        string
		args        args
		want        HAProxyState
		wantErr     *string
		ingressName string // The ingress to use for this test
	}{
		{
			name:        "delay with path regex",
			ingressName: "simple-ingress",
			args: args{
				in0: context.Background(),
				state: &HAProxyState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseDelay":        500,
						"conditionPathPattern": "/api/*",
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"simple-ingress"},
						},
					}),
				},
			},
			want: HAProxyState{
				ExecutionId:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				Namespace:        "demo",
				IngressName:      "simple-ingress",
				Matcher:          RequestMatcher{PathPattern: "/api/*"},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\ntcp-request inspect-delay 500ms\nacl sb_path_00000000_0000_0000_0000_000000000000 path_reg /api/*\ntcp-request content accept if WAIT_END || !sb_path_00000000_0000_0000_0000_000000000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name:        "delay with http method",
			ingressName: "simple-ingress",
			args: args{
				in0: context.Background(),
				state: &HAProxyState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseDelay":       500,
						"conditionHttpMethod": "POST",
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"simple-ingress"},
						},
					}),
				},
			},
			want: HAProxyState{
				ExecutionId:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				Namespace:        "demo",
				IngressName:      "simple-ingress",
				Matcher:          RequestMatcher{HttpMethod: "POST"},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\ntcp-request inspect-delay 500ms\nacl sb_method_00000000_0000_0000_0000_000000000000 method POST\ntcp-request content accept if WAIT_END || !sb_method_00000000_0000_0000_0000_000000000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name:        "delay with http header",
			ingressName: "simple-ingress",
			args: args{
				in0: context.Background(),
				state: &HAProxyState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseDelay": 500,
						"conditionHttpHeader": []interface{}{
							map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
						},
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"simple-ingress"},
						},
					}),
				},
			},
			want: HAProxyState{
				ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				Namespace:   "demo",
				IngressName: "simple-ingress",
				Matcher: RequestMatcher{HttpHeader: map[string]string{
					"User-Agent": "Mozilla.*",
				},
				},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\ntcp-request inspect-delay 500ms\nacl sb_hdr_User_Agent_00000000_0000_0000_0000_000000000000 hdr(User-Agent) -m reg Mozilla.*\ntcp-request content accept if WAIT_END || !sb_hdr_User_Agent_00000000_0000_0000_0000_000000000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name:        "delay with multiple conditions",
			ingressName: "simple-ingress",
			args: args{
				in0: context.Background(),
				state: &HAProxyState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseDelay":        1000,
						"conditionPathPattern": "/api/users",
						"conditionHttpMethod":  "POST",
						"conditionHttpHeader": []interface{}{
							map[string]interface{}{"key": "Content-Type", "value": "application/json"},
						},
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"simple-ingress"},
						},
					}),
				},
			},
			want: HAProxyState{
				ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				Namespace:   "demo",
				IngressName: "simple-ingress",
				Matcher: RequestMatcher{PathPattern: "/api/users",
					HttpMethod: "POST",
					HttpHeader: map[string]string{
						"Content-Type": "application/json",
					},
				},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\ntcp-request inspect-delay 1000ms\nacl sb_method_00000000_0000_0000_0000_000000000000 method POST\nacl sb_hdr_Content_Type_00000000_0000_0000_0000_000000000000 hdr(Content-Type) -m reg application/json\nacl sb_path_00000000_0000_0000_0000_000000000000 path_reg /api/users\ntcp-request content accept if WAIT_END || !sb_method_00000000_0000_0000_0000_000000000000 || !sb_hdr_Content_Type_00000000_0000_0000_0000_000000000000 || !sb_path_00000000_0000_0000_0000_000000000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name:        "no conditions provided",
			ingressName: "simple-ingress",
			args: args{
				in0: context.Background(),
				state: &HAProxyState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseDelay": 500,
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"simple-ingress"},
						},
					}),
				},
			},
			want:    HAProxyState{},
			wantErr: extutil.Ptr("at least one condition (path, method, or header) is required"),
		},
		{
			name:        "path collision with existing rule",
			ingressName: "path-conflict-ingress",
			args: args{
				in0: context.Background(),
				state: &HAProxyState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseDelay":        500,
						"conditionPathPattern": "/alreadyDelayed",
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"path-conflict-ingress"},
						},
					}),
				},
			},
			want:    HAProxyState{},
			wantErr: extutil.Ptr("a rule for path /alreadyDelayed already exists"),
		},
		{
			name:        "duplicate delay rule",
			ingressName: "delay-conflict-ingress",
			args: args{
				in0: context.Background(),
				state: &HAProxyState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseDelay":        500,
						"conditionPathPattern": "/newPath",
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"delay-conflict-ingress"},
						},
					}),
				},
			},
			want:    HAProxyState{},
			wantErr: extutil.Ptr("a delay rule already exists - cannot add another one"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update the target to use the correct ingress for this test
			if tt.args.request.Target != nil {
				tt.args.request.Target.Attributes["k8s.ingress"] = []string{tt.ingressName}
			}

			a := NewHAProxyDelayTrafficAction()
			state := a.NewEmptyState()
			_, err := a.Prepare(tt.args.in0, &state, tt.args.request)
			if tt.wantErr != nil {
				assert.EqualError(t, err, *tt.wantErr)
				return
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want.Matcher.PathPattern, state.Matcher.PathPattern)
			assert.Equal(t, tt.want.Matcher.HttpMethod, state.Matcher.HttpMethod)
			assert.Equal(t, tt.want.Matcher.HttpHeader, state.Matcher.HttpHeader)
			assert.Equal(t, tt.want.Namespace, state.Namespace)
			assert.Equal(t, tt.want.IngressName, state.IngressName)
			assert.Equal(t, tt.want.AnnotationConfig, state.AnnotationConfig)
		})
	}
}

func getTestClient(stopCh <-chan struct{}, objects ...runtime.Object) *client.Client {
	return client.CreateClient(testclient.NewClientset(objects...), stopCh, "", client.MockAllPermitted())
}
