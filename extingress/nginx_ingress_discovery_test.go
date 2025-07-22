/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extingress

import (
	"context"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"strings"
	"testing"
	"time"

	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_nginxIngressDiscovery_Basic(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	// setup client
	cli, cs := newTestClient(stopCh)
	extconfig.Config.ClusterName = "test-cluster"
	extconfig.Config.DisableDiscoveryExcludes = false

	// create NGINX IngressClasses dynamically
	class := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "nginxClass"},
		Spec:       networkingv1.IngressClassSpec{Controller: "k8s.io/ingress-nginx"},
	}
	_, err := cs.NetworkingV1().IngressClasses().Create(context.Background(), class, metav1.CreateOptions{})
	require.NoError(t, err)

	enterpriseClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "nginxEnterpriseClass"},
		Spec:       networkingv1.IngressClassSpec{Controller: "nginx.org/ingress-controller"},
	}
	_, err = cs.NetworkingV1().IngressClasses().Create(context.Background(), enterpriseClass, metav1.CreateOptions{})
	require.NoError(t, err)

	defaultClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "defaultNginxClass",
			Annotations: map[string]string{"ingressclass.kubernetes.io/is-default-class": "true"},
		},
		Spec: networkingv1.IngressClassSpec{Controller: "k8s.io/ingress-nginx"},
	}
	_, err = cs.NetworkingV1().IngressClasses().Create(context.Background(), defaultClass, metav1.CreateOptions{})
	require.NoError(t, err)

	// create Ingresses dynamically
	ing1 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ing1", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			IngressClassName: strPtr("nginxClass"),
			Rules:            []networkingv1.IngressRule{{Host: "host1.example.com"}},
		},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing1, metav1.CreateOptions{})
	require.NoError(t, err)

	ing2 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ing2", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			IngressClassName: strPtr("nginxEnterpriseClass"),
			Rules:            []networkingv1.IngressRule{{Host: "host2.example.com"}},
		},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing2, metav1.CreateOptions{})
	require.NoError(t, err)

	ing3 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ing3",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{Host: "host3.example.com"}},
		},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing3, metav1.CreateOptions{})
	require.NoError(t, err)

	ing4 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ing4", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			IngressClassName: strPtr("otherClass"), // Non-NGINX class
			Rules:            []networkingv1.IngressRule{{Host: "host4.example.com"}},
		},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing4, metav1.CreateOptions{})
	require.NoError(t, err)

	d := &nginxIngressDiscovery{k8s: cli}

	// Ensure only ingresses with NGINX classes are discovered
	assert.Eventually(t, func() bool {
		res, _ := d.DiscoverTargets(context.Background())
		return len(res) == 3 // Only ing1, ing2, and ing3 should be discovered
	}, time.Second, 100*time.Millisecond)

	targets, _ := d.DiscoverTargets(context.Background())
	require.Len(t, targets, 3) // Ensure three targets are discovered

	// Create a map for easier testing
	targetMap := make(map[string]discovery_kit_api.Target)
	for _, target := range targets {
		parts := strings.Split(target.Id, "/")
		require.Len(t, parts, 3)
		targetMap[parts[2]] = target
	}

	// Check ing1 (opensource nginx)
	if target, ok := targetMap["ing1"]; ok {
		assert.Equal(t, []string{"nginxClass"}, target.Attributes["k8s.ingress.class"])
		assert.Equal(t, []string{"k8s.io/ingress-nginx"}, target.Attributes["k8s.ingress.controller"])
		assert.Equal(t, []string{"host1.example.com"}, target.Attributes["k8s.ingress.hosts"])
	} else {
		t.Error("ing1 not found in discovered targets")
	}

	// Check ing2 (enterprise nginx)
	if target, ok := targetMap["ing2"]; ok {
		assert.Equal(t, []string{"nginxEnterpriseClass"}, target.Attributes["k8s.ingress.class"])
		assert.Equal(t, []string{"nginx.org/ingress-controller"}, target.Attributes["k8s.ingress.controller"])
		assert.Equal(t, []string{"host2.example.com"}, target.Attributes["k8s.ingress.hosts"])
	} else {
		t.Error("ing2 not found in discovered targets")
	}

	// Check ing3 (with annotation)
	if target, ok := targetMap["ing3"]; ok {
		assert.Equal(t, []string{"nginx"}, target.Attributes["k8s.ingress.class"])
		assert.Equal(t, []string{"host3.example.com"}, target.Attributes["k8s.ingress.hosts"])
	} else {
		t.Error("ing3 not found in discovered targets")
	}

	// Ensure ing4 is not discovered
	_, found := targetMap["ing4"]
	assert.False(t, found, "ing4 should not be discovered as it doesn't use a NGINX class")
}

func Test_nginxIngressDiscovery_ExcludeDisabled(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cli, cs := newTestClient(stopCh)
	extconfig.Config.ClusterName = "test-cluster"
	extconfig.Config.DisableDiscoveryExcludes = false

	// create class and disabled ingress
	class := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
		Spec:       networkingv1.IngressClassSpec{Controller: "k8s.io/ingress-nginx"},
	}
	_, err := cs.NetworkingV1().IngressClasses().Create(context.Background(), class, metav1.CreateOptions{})
	require.NoError(t, err)

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ignored",
			Namespace: "default",
			Labels:    map[string]string{"steadybit.com/discovery-disabled": "true"},
		},
		Spec: networkingv1.IngressSpec{IngressClassName: strPtr("nginx")},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing, metav1.CreateOptions{})
	require.NoError(t, err)

	d := &nginxIngressDiscovery{k8s: cli}

	assert.Eventually(t, func() bool {
		res, _ := d.DiscoverTargets(context.Background())
		return len(res) == 0
	}, time.Second, 100*time.Millisecond)
}

func Test_nginxIngressDiscovery_IncludeDisabledIfDisableDiscoveryExcludes(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cli, cs := newTestClient(stopCh)
	extconfig.Config.ClusterName = "test-cluster"
	extconfig.Config.DisableDiscoveryExcludes = true

	// create class and disabled ingress
	class := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
		Spec:       networkingv1.IngressClassSpec{Controller: "k8s.io/ingress-nginx"},
	}
	_, err := cs.NetworkingV1().IngressClasses().Create(context.Background(), class, metav1.CreateOptions{})
	require.NoError(t, err)

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "included",
			Namespace: "default",
			Labels:    map[string]string{"steadybit.com/discovery-disabled": "true"},
		},
		Spec: networkingv1.IngressSpec{IngressClassName: strPtr("nginx")},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing, metav1.CreateOptions{})
	require.NoError(t, err)

	d := &nginxIngressDiscovery{k8s: cli}

	assert.Eventually(t, func() bool {
		res, _ := d.DiscoverTargets(context.Background())
		return len(res) == 1
	}, time.Second, 100*time.Millisecond)

	targets, _ := d.DiscoverTargets(context.Background())
	assert.Equal(t, "test-cluster/default/included", targets[0].Id)
}



