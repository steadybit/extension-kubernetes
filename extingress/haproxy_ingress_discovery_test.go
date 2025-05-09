package extingress

import (
"context"
"strings"
"testing"
"time"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
networkingv1 "k8s.io/api/networking/v1"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
"k8s.io/apimachinery/pkg/runtime"
"k8s.io/client-go/kubernetes"
testclient "k8s.io/client-go/kubernetes/fake"

"github.com/steadybit/extension-kubernetes/v2/client"
"github.com/steadybit/extension-kubernetes/v2/extconfig"
)

func strPtr(s string) *string {
	return &s
}

// newTestClient creates a fake client with provided initial objects.
func newTestClient(stopCh <-chan struct{}, initObjs ...runtime.Object) (*client.Client, kubernetes.Interface) {
	cs := testclient.NewSimpleClientset(initObjs...)
	cli := client.CreateClient(cs, stopCh, "", client.MockAllPermitted())
	return cli, cs
}

func Test_ingressDiscovery_Basic(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	// setup client
	cli, cs := newTestClient(stopCh)
	extconfig.Config.ClusterName = "test-cluster"
	extconfig.Config.DisableDiscoveryExcludes = false

	// create HAProxy IngressClasses dynamically
	class := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "haproxyClass"},
		Spec:       networkingv1.IngressClassSpec{Controller: "haproxy.org/ingress-controller/haproxy"},
	}
	_, err := cs.NetworkingV1().IngressClasses().Create(context.Background(), class, metav1.CreateOptions{})
	require.NoError(t, err)
	defaultClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "defaultClass",
			Annotations: map[string]string{"ingressclass.kubernetes.io/is-default-class": "true"},
		},
		Spec: networkingv1.IngressClassSpec{Controller: "haproxy.org/ingress-controller/haproxy"},
	}
	_, err = cs.NetworkingV1().IngressClasses().Create(context.Background(), defaultClass, metav1.CreateOptions{})
	require.NoError(t, err)

	// create Ingresses dynamically
	ing1 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ing1", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			IngressClassName: strPtr("haproxyClass"),
			Rules:            []networkingv1.IngressRule{{Host: "host1"}},
		},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing1, metav1.CreateOptions{})
	require.NoError(t, err)
	ing2 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ing2", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			IngressClassName: strPtr("nginxClass"), // Non-HAProxy class
			Rules:            []networkingv1.IngressRule{{Host: "host2"}},
		},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing2, metav1.CreateOptions{})
	require.NoError(t, err)

	d := &ingressDiscovery{k8s: cli}

	// Ensure only ingresses with HAProxy classes are discovered
	assert.Eventually(t, func() bool {
		res, _ := d.DiscoverTargets(context.Background())
		return len(res) == 1 // Only ing1 should be discovered
	}, time.Second, 100*time.Millisecond)

	targets, _ := d.DiscoverTargets(context.Background())
	require.Len(t, targets, 1) // Ensure only one target is discovered
	for _, target := range targets {
		parts := strings.Split(target.Id, "/")
		require.Len(t, parts, 3)
		switch parts[2] {
		case "ing1":
			assert.Equal(t, []string{"haproxyClass"}, target.Attributes["k8s.ingress.class"])
			assert.Equal(t, []string{"haproxy.org/ingress-controller/haproxy"}, target.Attributes["k8s.ingress.controller"])
			assert.Equal(t, []string{"host1"}, target.Attributes["k8s.ingress.hosts"])
		default:
			t.Errorf("unexpected ingress: %s", parts[2])
		}
	}
}


func Test_ingressDiscovery_ExcludeDisabled(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cli, cs := newTestClient(stopCh)
	extconfig.Config.ClusterName = "test-cluster"
	extconfig.Config.DisableDiscoveryExcludes = false

	// create class and disabled ingress
	class := &networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "haproxy"}, Spec: networkingv1.IngressClassSpec{Controller: "haproxy.org/ingress-controller/haproxy"}}
	_, err := cs.NetworkingV1().IngressClasses().Create(context.Background(), class, metav1.CreateOptions{})
	require.NoError(t, err)
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ignored", Namespace: "default", Labels: map[string]string{"steadybit.com/discovery-disabled": "true"}},
		Spec: networkingv1.IngressSpec{IngressClassName: strPtr("haproxy")},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing, metav1.CreateOptions{})
	require.NoError(t, err)

	d := &ingressDiscovery{k8s: cli}

	assert.Eventually(t, func() bool {
		res, _ := d.DiscoverTargets(context.Background())
		return len(res) == 0
	}, time.Second, 100*time.Millisecond)
}

func Test_ingressDiscovery_IncludeDisabledIfDisableDiscoveryExcludes(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cli, cs := newTestClient(stopCh)
	extconfig.Config.ClusterName = "test-cluster"
	extconfig.Config.DisableDiscoveryExcludes = true

	// create class and disabled ingress
	class := &networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "haproxy"}, Spec: networkingv1.IngressClassSpec{Controller: "haproxy.org/ingress-controller/haproxy"}}
	_, err := cs.NetworkingV1().IngressClasses().Create(context.Background(), class, metav1.CreateOptions{})
	require.NoError(t, err)
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "included", Namespace: "default", Labels: map[string]string{"steadybit.com/discovery-disabled": "true"}},
		Spec: networkingv1.IngressSpec{IngressClassName: strPtr("haproxy")},
	}
	_, err = cs.NetworkingV1().Ingresses("default").Create(context.Background(), ing, metav1.CreateOptions{})
	require.NoError(t, err)

	d := &ingressDiscovery{k8s: cli}

	assert.Eventually(t, func() bool {
		res, _ := d.DiscoverTargets(context.Background())
		return len(res) == 1
	}, time.Second, 100*time.Millisecond)

	targets, _ := d.DiscoverTargets(context.Background())
	assert.Equal(t, "test-cluster/default/included", targets[0].Id)
}
