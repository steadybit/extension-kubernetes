package extingress

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func TestHAProxyBlockTrafficAction_Prepare(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	testClient, clientset := getTestClient(stopCh)
	_, err := clientset.
		NetworkingV1().
		Ingresses("demo").
		Create(context.Background(), &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-ingress",
				Namespace: "demo",
				Annotations: map[string]string{
					"kubernetes.io/ingress.class": "haproxy",
					AnnotationKey:                 "http-request return status 503 if sb_path_abcd path_reg /alreadyBlocked",
				},
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)
	client.K8S = testClient
	assert.Eventually(t, func() bool {
		name, _ := testClient.IngressByNamespaceAndName("demo", "my-ingress")
		return name != nil
	}, time.Second, 100*time.Millisecond)

	type args struct {
		in0     context.Context
		state   *HAProxyBlockTrafficState
		request action_kit_api.PrepareActionRequestBody
	}
	tests := []struct {
		name    string
		args    args
		want    HAProxyBlockTrafficState
		wantErr *string
	}{
		{
			name: "block with path regex",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseStatusCode":   503,
						"conditionPathPattern": "/api/*",
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			want: HAProxyBlockTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Namespace:   "demo",
					IngressName: "my-ingress",
				},
				ResponseStatusCode:   503,
				ConditionPathPattern: "/api/*",
				AnnotationConfig:     "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nacl sb_path_00000000 path_reg /api/*\nhttp-request return status 503 if sb_path_00000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name: "block with http method",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseStatusCode":  503,
						"conditionHttpMethod": "POST",
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			want: HAProxyBlockTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Namespace:   "demo",
					IngressName: "my-ingress",
				},
				ResponseStatusCode:  503,
				ConditionHttpMethod: "POST",
				AnnotationConfig:    "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nacl sb_method_00000000 method POST\nhttp-request return status 503 if sb_method_00000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name: "block with http header",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseStatusCode": 503,
						"conditionHttpHeader": map[string]interface{}{
							"User-Agent": "Mozilla.*",
						},
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			want: HAProxyBlockTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Namespace:   "demo",
					IngressName: "my-ingress",
				},
				ResponseStatusCode: 503,
				ConditionHttpHeader: map[string]string{
					"User-Agent": "Mozilla.*",
				},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nacl sb_hdr_User_Agent_00000000 hdr(User-Agent) -m reg Mozilla.*\nhttp-request return status 503 if sb_hdr_User_Agent_00000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name: "block with multiple conditions",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseStatusCode":   503,
						"conditionPathPattern": "/api/users",
						"conditionHttpMethod":  "POST",
						"conditionHttpHeader": map[string]interface{}{
							"Content-Type": "application/json",
						},
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			want: HAProxyBlockTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Namespace:   "demo",
					IngressName: "my-ingress",
				},
				ResponseStatusCode:   503,
				ConditionPathPattern: "/api/users",
				ConditionHttpMethod:  "POST",
				ConditionHttpHeader: map[string]string{
					"Content-Type": "application/json",
				},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nacl sb_method_00000000 method POST\nacl sb_hdr_Content_Type_00000000 hdr(Content-Type) -m reg application/json\nacl sb_path_00000000 path_reg /api/users\nhttp-request return status 503 if sb_method_00000000 sb_hdr_Content_Type_00000000 sb_path_00000000\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name: "no conditions provided",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseStatusCode": 503,
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			want:    HAProxyBlockTrafficState{},
			wantErr: extutil.Ptr("at least one condition is required"),
		},
		{
			name: "path collision with existing rule",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Config: map[string]interface{}{
						"responseStatusCode":   503,
						"conditionPathPattern": "/alreadyBlocked",
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			want:    HAProxyBlockTrafficState{},
			wantErr: extutil.Ptr("a rule for path /alreadyBlocked already exists"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &HAProxyBlockTrafficAction{}
			state := a.NewEmptyState()
			_, err := a.Prepare(tt.args.in0, &state, tt.args.request)
			if tt.wantErr != nil {
				assert.EqualError(t, err, *tt.wantErr)
				return
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want.ResponseStatusCode, state.ResponseStatusCode)
			assert.Equal(t, tt.want.ConditionPathPattern, state.ConditionPathPattern)
			assert.Equal(t, tt.want.ConditionHttpMethod, state.ConditionHttpMethod)
			assert.Equal(t, tt.want.ConditionHttpHeader, state.ConditionHttpHeader)
			assert.Equal(t, tt.want.Namespace, state.Namespace)
			assert.Equal(t, tt.want.IngressName, state.IngressName)
			// We won't compare the exact annotation config string since the ACL names contain
			// random elements from the UUID, but we can verify it contains the essential parts
			if tt.want.AnnotationConfig != "" {
				assert.Contains(t, state.AnnotationConfig, "# BEGIN STEADYBIT")
				assert.Contains(t, state.AnnotationConfig, "# END STEADYBIT")
				assert.Contains(t, state.AnnotationConfig, fmt.Sprintf("http-request return status %d", state.ResponseStatusCode))

				if state.ConditionPathPattern != "" {
					assert.Contains(t, state.AnnotationConfig, "path_reg")
					assert.Contains(t, state.AnnotationConfig, state.ConditionPathPattern)
				}
				if state.ConditionHttpMethod != "" {
					assert.Contains(t, state.AnnotationConfig, "method")
					assert.Contains(t, state.AnnotationConfig, state.ConditionHttpMethod)
				}
				for headerName, headerValue := range state.ConditionHttpHeader {
					assert.Contains(t, state.AnnotationConfig, headerName)
					assert.Contains(t, state.AnnotationConfig, headerValue)
				}
			}
		})
	}
}

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	client := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())
	return client, clientset
}
