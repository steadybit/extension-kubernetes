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
					AnnotationKey:                 "http-request return status 503 if { path_reg /alreadyBlocked }",
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
			name: "valid pathStatusCode",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.New(),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					Config: map[string]interface{}{
						"pathStatusCode": []interface{}{
							map[string]interface{}{"key": "/block", "value": "502"},
							map[string]interface{}{"key": "/block2", "value": "503"},
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
				PathStatusCode: map[string]int{
					"/block":  502,
					"/block2": 503,
				},
				AnnotationConfig: "# BEGIN STEADYBIT - 00000000-0000-0000-0000-000000000000\nhttp-request return status 502 if { path_reg /block }\nhttp-request return status 503 if { path_reg /block2 }\n# END STEADYBIT - 00000000-0000-0000-0000-000000000000\n",
			},
			wantErr: nil,
		},
		{
			name: "invalid status code",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.New(),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					Config: map[string]interface{}{
						"pathStatusCode": []interface{}{
							map[string]interface{}{"key": "/block", "value": "abc"},
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
			want:    HAProxyBlockTrafficState{},
			wantErr: extutil.Ptr("invalid status code: abc"),
		},
		{
			name: "duplicate path rule",
			args: args{
				in0: context.Background(),
				state: &HAProxyBlockTrafficState{
					HAProxyBaseState: HAProxyBaseState{
						ExecutionId: uuid.New(),
					},
				},
				request: action_kit_api.PrepareActionRequestBody{
					Config: map[string]interface{}{
						"pathStatusCode": []interface{}{
							map[string]interface{}{"key": "/block", "value": "503"},
							map[string]interface{}{"key": "/alreadyBlocked", "value": "503"},
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
			want:    HAProxyBlockTrafficState{},
			wantErr: extutil.Ptr("a rule for path /alreadyBlocked already exists"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &HAProxyBlockTrafficAction{}
			action := NewHAProxyBlockTrafficAction()
			state := action.NewEmptyState()
			_, err := a.Prepare(tt.args.in0, &state, tt.args.request)
			if tt.wantErr != nil {
				assert.EqualError(t, err, *tt.wantErr)
				return
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.PathStatusCode, state.PathStatusCode)
			assert.NotNil(t, state)
			assert.Equalf(t, tt.want.PathStatusCode, state.PathStatusCode, "PathStatusCode")
			assert.Equalf(t, tt.want.AnnotationConfig, state.AnnotationConfig, "AnnotationConfig")
			assert.Equalf(t, tt.want.Namespace, state.Namespace, "Namespace")
			assert.Equalf(t, tt.want.IngressName, state.IngressName, "IngressName")
			assert.Equalf(t, tt.want.ExecutionId, state.ExecutionId, "ExecutionId")
		})
	}
}

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	client := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())
	return client, clientset
}
