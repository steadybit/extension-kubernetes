// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extingress

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestParseHAProxyImageVersion(t *testing.T) {
	tests := []struct {
		image     string
		wantMajor int
		wantMinor int
		wantOk    bool
	}{
		{image: "haproxytech/kubernetes-ingress:3.2.19", wantMajor: 3, wantMinor: 2, wantOk: true},
		{image: "docker.io/haproxytech/kubernetes-ingress:3.4.0", wantMajor: 3, wantMinor: 4, wantOk: true},
		{image: "my-registry:5000/haproxytech/kubernetes-ingress:v3.4.1", wantMajor: 3, wantMinor: 4, wantOk: true},
		{image: "haproxytech/kubernetes-ingress:3.4.0@sha256:abc", wantMajor: 3, wantMinor: 4, wantOk: true},
		{image: "haproxytech/kubernetes-ingress@sha256:abc", wantOk: false},
		{image: "haproxytech/kubernetes-ingress:latest", wantOk: false},
		{image: "haproxytech/kubernetes-ingress", wantOk: false},
	}
	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			major, minor, ok := parseHAProxyImageVersion(tt.image)
			assert.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				assert.Equal(t, tt.wantMajor, major)
				assert.Equal(t, tt.wantMinor, minor)
			}
		})
	}
}

func TestIsHAProxyIngressControllerImage(t *testing.T) {
	assert.True(t, isHAProxyIngressControllerImage("haproxytech/kubernetes-ingress:3.4.0"))
	assert.True(t, isHAProxyIngressControllerImage("my-registry:5000/haproxytech/kubernetes-ingress:3.4.0"))
	assert.False(t, isHAProxyIngressControllerImage("haproxy:3.4"))
	assert.False(t, isHAProxyIngressControllerImage("registry.k8s.io/ingress-nginx/controller:v1.12.0"))
}

func TestFindHAProxyControllerPods(t *testing.T) {
	pod32 := haProxyControllerPod("haproxy-32", "3.2.19", "--ingress.class=haproxy")
	pod34 := haProxyControllerPod("haproxy-34", "3.4.0", "--ingress.class=haproxy-new")
	podNoClass := haProxyControllerPod("haproxy-any", "3.4.0")
	otherPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "demo"},
		Status:     corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: "app", Image: "nginx:1.27"}}},
	}

	t.Run("matches pods by ingress class", func(t *testing.T) {
		pods := findHAProxyControllerPods([]*corev1.Pod{pod32, pod34, podNoClass, otherPod}, "haproxy")
		assert.Equal(t, []*corev1.Pod{pod32}, pods)
	})

	t.Run("falls back to all controller pods if no pod declares the class", func(t *testing.T) {
		pods := findHAProxyControllerPods([]*corev1.Pod{pod32, pod34, podNoClass, otherPod}, "unknown")
		assert.Equal(t, []*corev1.Pod{pod32, pod34, podNoClass}, pods)
	})
}

func TestHAProxyAction_Prepare_ControllerVersionCheck(t *testing.T) {
	tests := []struct {
		name        string
		pods        []runtime.Object
		wantErr     bool
		errContains string
	}{
		{
			name:    "controller 3.2 is supported",
			pods:    []runtime.Object{haProxyControllerPod("haproxy", "3.2.19", "--ingress.class=haproxy")},
			wantErr: false,
		},
		{
			name:        "controller 3.4 is not supported",
			pods:        []runtime.Object{haProxyControllerPod("haproxy", "3.4.0", "--ingress.class=haproxy")},
			wantErr:     true,
			errContains: "HAProxy ingress controller 3.4 (pod haproxy-controller/haproxy) is not supported yet",
		},
		{
			name:    "unknown controller version is allowed",
			pods:    []runtime.Object{haProxyControllerPod("haproxy", "latest", "--ingress.class=haproxy")},
			wantErr: false,
		},
		{
			name:    "no controller pod visible is allowed",
			pods:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopCh := make(chan struct{})
			defer close(stopCh)

			objects := append([]runtime.Object{
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{Name: "test-ingress", Namespace: "demo"},
				},
			}, tt.pods...)
			testClient := getTestClient(stopCh, objects...)
			client.K8S = testClient

			require.Eventually(t, func() bool {
				ingress, _ := testClient.IngressByNamespaceAndName("demo", "test-ingress")
				return ingress != nil && len(testClient.Pods()) == len(tt.pods)
			}, time.Second, 100*time.Millisecond)

			action := NewHAProxyDelayTrafficAction()
			state := action.NewEmptyState()
			request := action_kit_api.PrepareActionRequestBody{
				Config: map[string]any{
					"duration":             10000,
					"responseDelay":        500,
					"conditionPathPattern": "/test",
				},
				Target: &action_kit_api.Target{
					Attributes: map[string][]string{
						"k8s.namespace":     {"demo"},
						"k8s.ingress":       {"test-ingress"},
						"k8s.ingress.class": {"haproxy"},
					},
				},
				ExecutionId: uuid.New(),
			}

			_, err := action.Prepare(context.Background(), &state, request)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func haProxyControllerPod(name, tag string, args ...string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "haproxy-controller"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "kubernetes-ingress-controller",
				Args: args,
			}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "kubernetes-ingress-controller",
				Image: "haproxytech/kubernetes-ingress:" + tag,
			}},
		},
	}
}
