package extingress

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestHAProxyDelayTrafficAction_Prepare(t *testing.T) {
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
		state   *HAProxyDelayTrafficState
		request action_kit_api.PrepareActionRequestBody
	}
	tests := []struct {
		name    string
		args    args
		existingIngressAnnotation *string
		want    HAProxyDelayTrafficState
		wantErr *string
	}{
		{
			name: "valid pathDelay",
			args: args{
				in0: context.Background(),
				state: &HAProxyDelayTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.New(),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					Config: map[string]interface{}{
						"path":  "/delay",
						"delay": 1000,
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			existingIngressAnnotation: nil,
			want: HAProxyDelayTrafficState{
				HAProxyBaseState: HAProxyBaseState{
					ExecutionId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					Namespace:   "demo",
					IngressName: "my-ingress",
				},
				Path:             "/delay",
				Delay:            1000,
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\ntcp-request inspect-delay 1000ms\ntcp-request content accept if WAIT_END || !{ path_reg /delay }\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name: "invalid delay value",
			args: args{
				in0: context.Background(),
				state: &HAProxyDelayTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.New(),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					Config: map[string]interface{}{
						"path":  "/delay",
						"delay": "abc",
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			existingIngressAnnotation: nil,
			want:    HAProxyDelayTrafficState{},
			wantErr: extutil.Ptr("delay must be a number, got string: abc"),
		},
		{
			name: "duplicate path rule",
			args: args{
				in0: context.Background(),
				state: &HAProxyDelayTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.New(),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					Config: map[string]interface{}{
						"path":  "/alreadydelay",
						"delay": 1000,
					},
					Target: extutil.Ptr(action_kit_api.Target{
						Attributes: map[string][]string{
							"k8s.namespace": {"demo"},
							"k8s.ingress":   {"my-ingress"},
						},
					}),
				},
			},
			existingIngressAnnotation: extutil.Ptr("tcp-request inspect-delay 1000ms\ntcp-request content accept if WAIT_END || !{ path_reg /alreadydelay }"),
			want:    HAProxyDelayTrafficState{},
			wantErr: extutil.Ptr("a delay rule already exists - cannot add another one"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.existingIngressAnnotation != nil {
				_, err := clientset.
					NetworkingV1().
					Ingresses("demo").
					Update(context.Background(), &networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "my-ingress",
							Namespace:   "demo",
							Annotations: map[string]string{AnnotationKey: *tt.existingIngressAnnotation},
						},
					}, metav1.UpdateOptions{})
				require.NoError(t, err)
			} else {
				_, err := clientset.
					NetworkingV1().
					Ingresses("demo").
					Update(context.Background(), &networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-ingress",
							Namespace: "demo",
						},
					}, metav1.UpdateOptions{})
				require.NoError(t, err)
			}

			a := &HAProxyDelayTrafficAction{}
			action := NewHAProxyDelayTrafficAction()
			state := action.NewEmptyState()
			_, err := a.Prepare(tt.args.in0, &state, tt.args.request)
			if tt.wantErr != nil {
				assert.EqualError(t, err, *tt.wantErr)
				return
			} else {
				require.NoError(t, err)
			}
			assert.NotNil(t, state)
			assert.Equalf(t, tt.want.Path, state.Path, "Path")
			assert.Equalf(t, tt.want.Delay, state.Delay, "Delay")
			assert.Equalf(t, tt.want.AnnotationConfig, state.AnnotationConfig, "AnnotationConfig")
			assert.Equalf(t, tt.want.Namespace, state.Namespace, "Namespace")
			assert.Equalf(t, tt.want.IngressName, state.IngressName, "IngressName")
			assert.Equalf(t, tt.want.ExecutionId, state.ExecutionId, "ExecutionId")
		})
	}
}
