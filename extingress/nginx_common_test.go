/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extingress

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/steadybit/extension-kubernetes/v2/client"
)

// newNginxTestClient creates a fake client with provided initial objects.
func newNginxTestClient(stopCh <-chan struct{}, initObjs ...runtime.Object) (*client.Client, kubernetes.Interface) {
	cs := testclient.NewClientset(initObjs...)
	cli := client.CreateClient(cs, stopCh, "", client.MockAllPermitted())
	return cli, cs
}

func Test_findNginxControllerNamespace_Basic(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cli, cs := newNginxTestClient(stopCh)

	// Override the global client for testing
	originalClient := client.K8S
	client.K8S = cli
	defer func() { client.K8S = originalClient }()

	// Test 1: Non-NGINX controller should return empty
	nonNginxClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "traefik"},
		Spec:       networkingv1.IngressClassSpec{Controller: "traefik.io/ingress-controller"},
	}
	_, err := cs.NetworkingV1().IngressClasses().Create(context.Background(), nonNginxClass, metav1.CreateOptions{})
	require.NoError(t, err)

	// Test 2: Valid NGINX controller
	nginxClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
		Spec:       networkingv1.IngressClassSpec{Controller: "k8s.io/ingress-nginx"},
	}
	_, err = cs.NetworkingV1().IngressClasses().Create(context.Background(), nginxClass, metav1.CreateOptions{})
	require.NoError(t, err)

	// Wait for IngressClass to be available in cache
	assert.Eventually(t, func() bool {
		classes := client.K8S.IngressClasses()
		return len(classes) >= 2
	}, time.Second*3, 100*time.Millisecond)

	// Test non-NGINX controller
	result := findNginxControllerNamespace("traefik")
	assert.Equal(t, "", result, "Non-NGINX controller should return empty")

	// Test NGINX controller - will return empty since no pods exist, but should identify as NGINX
	result = findNginxControllerNamespace("nginx")
	// This will be empty because no pods exist, but that's expected behavior
	assert.Equal(t, "", result, "NGINX controller without pods should return empty")

	// Test non-existent class
	result = findNginxControllerNamespace("nonexistent")
	assert.Equal(t, "", result, "Non-existent controller should return empty")
}

func Test_isNginxController(t *testing.T) {
	tests := []struct {
		controller string
		expected   bool
	}{
		{"k8s.io/ingress-nginx", true},
		{"nginx.org/ingress-controller", true},
		{"traefik.io/ingress-controller", false},
		{"haproxy.org/ingress-controller/haproxy", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.controller, func(t *testing.T) {
			result := isNginxController(tt.controller)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_hasNginxControllerPods(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cli, _ := newNginxTestClient(stopCh)

	// Override the global client for testing
	originalClient := client.K8S
	client.K8S = cli
	defer func() { client.K8S = originalClient }()

	// Test with non-existent namespace
	result := hasNginxControllerPods("non-existent")
	assert.False(t, result, "Non-existent namespace should return false")

	// Test with existing namespace but no pods
	result = hasNginxControllerPods("default")
	assert.False(t, result, "Namespace without NGINX pods should return false")
}

func Test_findNginxControllerNamespace_WithAnnotations(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cli, cs := newNginxTestClient(stopCh)

	// Override the global client for testing
	originalClient := client.K8S
	client.K8S = cli
	defer func() { client.K8S = originalClient }()

	// Test UBI NGINX with operator-sdk/primary-resource annotation
	ubiNginxClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ubi-nginx",
			Annotations: map[string]string{
				"operator-sdk/primary-resource": "nginx-ingress-steadybit/nginxingress-controller",
			},
		},
		Spec: networkingv1.IngressClassSpec{Controller: "nginx.org/ingress-controller"},
	}
	_, err := cs.NetworkingV1().IngressClasses().Create(context.Background(), ubiNginxClass, metav1.CreateOptions{})
	require.NoError(t, err)

	// Test community NGINX with meta.helm.sh/release-namespace annotation
	communityNginxClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "community-nginx",
			Annotations: map[string]string{
				"meta.helm.sh/release-namespace": "ingress-nginx",
			},
		},
		Spec: networkingv1.IngressClassSpec{Controller: "k8s.io/ingress-nginx"},
	}
	_, err = cs.NetworkingV1().IngressClasses().Create(context.Background(), communityNginxClass, metav1.CreateOptions{})
	require.NoError(t, err)

	// Wait for IngressClasses to be available in cache
	assert.Eventually(t, func() bool {
		classes := client.K8S.IngressClasses()
		return len(classes) >= 2
	}, time.Second*3, 100*time.Millisecond)

	// Test UBI NGINX - should try to look in nginx-ingress-steadybit namespace
	result := findNginxControllerNamespace("ubi-nginx")
	// Will return empty since no pods exist, but that's expected in test environment
	assert.Equal(t, "", result, "UBI NGINX controller without pods should return empty")

	// Test community NGINX - should try to look in ingress-nginx namespace
	result = findNginxControllerNamespace("community-nginx")
	// Will return empty since no pods exist, but that's expected in test environment
	assert.Equal(t, "", result, "Community NGINX controller without pods should return empty")
}

func Test_podServesIngressClass(t *testing.T) {
	tests := []struct {
		name         string
		pod          *corev1.Pod
		ingressClass string
		expected     bool
	}{
		{
			name: "pod with -ingress-class flag (separate args)",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "controller",
							Args: []string{"/nginx-ingress-controller", "-ingress-class", "nginx"},
						},
					},
				},
			},
			ingressClass: "nginx",
			expected:     true,
		},
		{
			name: "pod with --ingress-class flag (separate args)",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "controller",
							Args: []string{"/nginx-ingress-controller", "--ingress-class", "nginx"},
						},
					},
				},
			},
			ingressClass: "nginx",
			expected:     true,
		},
		{
			name: "pod with -ingress-class=value format",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "controller",
							Args: []string{"/nginx-ingress-controller", "-ingress-class=nginx"},
						},
					},
				},
			},
			ingressClass: "nginx",
			expected:     true,
		},
		{
			name: "pod with --ingress-class=value format",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "controller",
							Args: []string{"/nginx-ingress-controller", "--ingress-class=nginx"},
						},
					},
				},
			},
			ingressClass: "nginx",
			expected:     true,
		},
		{
			name: "pod with wrong ingress class",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "controller",
							Args: []string{"/nginx-ingress-controller", "-ingress-class", "different"},
						},
					},
				},
			},
			ingressClass: "nginx",
			expected:     false,
		},
		{
			name: "pod without ingress class args",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "controller",
							Args: []string{"/nginx-ingress-controller", "-configmap=nginx/config"},
						},
					},
				},
			},
			ingressClass: "nginx",
			expected:     false,
		},
		{
			name: "pod with multiple containers, one matches",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "sidecar",
							Args: []string{"/sidecar"},
						},
						{
							Name: "controller",
							Args: []string{"/nginx-ingress-controller", "-ingress-class=nginx"},
						},
					},
				},
			},
			ingressClass: "nginx",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := podServesIngressClass(tt.pod, tt.ingressClass)
			assert.Equal(t, tt.expected, result)
		})
	}
}
