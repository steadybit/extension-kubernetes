// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// --- Fault spec builders -----------------------------------------------------

func Test_buildDelayFaultSpec(t *testing.T) {
	spec, err := buildDelayFaultSpec(map[string]any{"delay": float64(5000), "percentage": float64(25)})
	require.NoError(t, err)

	delay := spec["faultInjection"].(map[string]any)["delay"].(map[string]any)
	assert.Equal(t, "5000ms", delay["fixedDelay"])
	assert.Equal(t, float64(25), delay["percentage"])

	_, err = buildDelayFaultSpec(map[string]any{"delay": float64(0)})
	assert.Error(t, err)
}

func Test_buildStatusFaultSpec(t *testing.T) {
	spec, err := buildStatusFaultSpec(map[string]any{"statusCode": float64(503), "percentage": float64(50)})
	require.NoError(t, err)

	abort := spec["faultInjection"].(map[string]any)["abort"].(map[string]any)
	assert.Equal(t, int64(503), abort["httpStatus"])
	assert.Equal(t, float64(50), abort["percentage"])

	_, err = buildStatusFaultSpec(map[string]any{"statusCode": float64(700)})
	assert.Error(t, err)
}

func Test_buildResponseBodyFaultSpec(t *testing.T) {
	spec, err := buildResponseBodyFaultSpec(map[string]any{
		"statusCode":  float64(200),
		"body":        `{"error":"chaos"}`,
		"contentType": "application/json",
		"percentage":  float64(10),
		// sentinelStatus unset -> defaults to 418
	})
	require.NoError(t, err)

	abort := spec["faultInjection"].(map[string]any)["abort"].(map[string]any)
	assert.Equal(t, int64(418), abort["httpStatus"])
	assert.Equal(t, float64(10), abort["percentage"])

	override := spec["responseOverride"].([]any)[0].(map[string]any)
	assert.Equal(t, "Local", override["source"])
	statusCodes := override["match"].(map[string]any)["statusCodes"].([]any)[0].(map[string]any)
	assert.Equal(t, "Value", statusCodes["type"])
	assert.Equal(t, int64(418), statusCodes["value"])
	response := override["response"].(map[string]any)
	assert.Equal(t, int64(200), response["statusCode"])
	assert.Equal(t, "application/json", response["contentType"])
	body := response["body"].(map[string]any)
	assert.Equal(t, "Inline", body["type"])
	assert.Equal(t, `{"error":"chaos"}`, body["inline"])
}

func Test_buildResponseBodyFaultSpec_validations(t *testing.T) {
	_, err := buildResponseBodyFaultSpec(map[string]any{"statusCode": float64(200), "body": ""})
	assert.Error(t, err, "empty body should fail")

	_, err = buildResponseBodyFaultSpec(map[string]any{"statusCode": float64(418), "body": "x", "sentinelStatus": float64(418)})
	assert.Error(t, err, "sentinel equal to final status should fail")
}

func Test_percentageFromConfig_defaultsTo100(t *testing.T) {
	assert.Equal(t, float64(100), percentageFromConfig(map[string]any{}))
	assert.Equal(t, float64(30), percentageFromConfig(map[string]any{"percentage": float64(30)}))
	assert.Equal(t, float64(30), percentageFromConfig(map[string]any{"percentage": 30}))
}

// --- Conflict detection ------------------------------------------------------

func policyTargeting(name, routeName, sectionName string) unstructured.Unstructured {
	ref := map[string]any{"group": gatewayAPIGroup, "kind": httpRouteKind, "name": routeName}
	if sectionName != "" {
		ref["sectionName"] = sectionName
	}
	return unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": name},
		"spec":     map[string]any{"targetRefs": []any{ref}},
	}}
}

func Test_findConflictingPolicy(t *testing.T) {
	existing := []unstructured.Unstructured{policyTargeting("other", "shop", "")}

	// Whole-route conflict.
	assert.Equal(t, "other", findConflictingPolicy(existing, "shop", "", "mine"))
	// Different route -> no conflict.
	assert.Equal(t, "", findConflictingPolicy(existing, "cart", "", "mine"))
	// Our own policy is ignored.
	assert.Equal(t, "", findConflictingPolicy([]unstructured.Unstructured{policyTargeting("mine", "shop", "")}, "shop", "", "mine"))

	// Section-scoped existing policy only conflicts with same section.
	sectioned := []unstructured.Unstructured{policyTargeting("sec", "shop", "rule-a")}
	assert.Equal(t, "sec", findConflictingPolicy(sectioned, "shop", "rule-a", "mine"))
	assert.Equal(t, "", findConflictingPolicy(sectioned, "shop", "rule-b", "mine"))
	// A whole-route attack conflicts with a section-scoped existing policy.
	assert.Equal(t, "sec", findConflictingPolicy(sectioned, "shop", "", "mine"))
}

func Test_buildBackendTrafficPolicy(t *testing.T) {
	policy := buildBackendTrafficPolicy("default", "steadybit-delay-abc", "abc", "shop", "rule-a", map[string]any{
		"faultInjection": map[string]any{"delay": map[string]any{"fixedDelay": "5s"}},
	})

	assert.Equal(t, btpAPIVersion, policy.GetAPIVersion())
	assert.Equal(t, btpKind, policy.GetKind())
	assert.Equal(t, "steadybit-delay-abc", policy.GetName())
	assert.Equal(t, "default", policy.GetNamespace())
	assert.Equal(t, managedByValue, policy.GetLabels()[managedByLabelKey])
	assert.Equal(t, "abc", policy.GetLabels()[executionLabelKey])

	targetRefs, _, _ := unstructured.NestedSlice(policy.Object, "spec", "targetRefs")
	ref := targetRefs[0].(map[string]any)
	assert.Equal(t, "shop", ref["name"])
	assert.Equal(t, "rule-a", ref["sectionName"])
}

// --- Discovery ---------------------------------------------------------------

var (
	httpRouteGVK    = schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"}
	gatewayGVK      = schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway"}
	gatewayClassGVK = schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GatewayClass"}
	btpGVK          = schema.GroupVersionKind{Group: "gateway.envoyproxy.io", Version: "v1alpha1", Kind: "BackendTrafficPolicy"}
)

func getTestClient(stopCh <-chan struct{}) (*client.Client, dynamic.Interface) {
	extconfig.Config.DiscoveryDisabledEnvoyGateway = false
	// The global config struct is zero-valued in tests; the real "disabled by default" comes from the
	// envconfig tag, so explicitly disable Argo Rollouts here to avoid its informer panicking on a
	// dynamic client that doesn't register the rollout list kind.
	extconfig.Config.DiscoveryDisabledArgoRollout = true
	extconfig.Config.ClusterName = "test-cluster"

	scheme := runtime.NewScheme()
	for _, gvk := range []schema.GroupVersionKind{httpRouteGVK, gatewayGVK, gatewayClassGVK, btpGVK} {
		scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		scheme.AddKnownTypeWithName(listGVK(gvk), &unstructured.UnstructuredList{})
	}
	// The fake client naively pluralizes kinds (Gateway -> gatewaies), so we must map each real
	// resource name to its list kind explicitly.
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		client.HTTPRouteGVR:            "HTTPRouteList",
		client.GatewayGVR:              "GatewayList",
		client.GatewayClassGVR:         "GatewayClassList",
		client.BackendTrafficPolicyGVR: "BackendTrafficPolicyList",
	})

	clientset := testclient.NewSimpleClientset()
	k8sClient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)
	k8sClient.Distribution = "kubernetes"
	return k8sClient, dynamicClient
}

func listGVK(gvk schema.GroupVersionKind) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind + "List"}
}

func gatewayClass(name, controller string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "gateway.networking.k8s.io/v1", "kind": "GatewayClass",
		"metadata": map[string]any{"name": name},
		"spec":     map[string]any{"controllerName": controller},
	}}
}

func gateway(namespace, name, className string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "gateway.networking.k8s.io/v1", "kind": "Gateway",
		"metadata": map[string]any{"name": name, "namespace": namespace},
		"spec":     map[string]any{"gatewayClassName": className},
	}}
}

func httpRoute(namespace, name, gatewayName string, hostnames []string) *unstructured.Unstructured {
	hn := make([]any, len(hostnames))
	for i, h := range hostnames {
		hn[i] = h
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "gateway.networking.k8s.io/v1", "kind": "HTTPRoute",
		"metadata": map[string]any{"name": name, "namespace": namespace},
		"spec": map[string]any{
			"parentRefs": []any{map[string]any{"name": gatewayName}},
			"hostnames":  hn,
		},
	}}
}

func create(t *testing.T, dc dynamic.Interface, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) {
	t.Helper()
	_, err := dc.Resource(gvr).Namespace(obj.GetNamespace()).Create(context.Background(), obj, metav1.CreateOptions{})
	require.NoError(t, err)
}

func Test_httpRouteDiscovery_filtersByEnvoyGatewayClass(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sClient, dc := getTestClient(stopCh)

	create(t, dc, client.GatewayClassGVR, gatewayClass("eg", envoyGatewayControllerName))
	create(t, dc, client.GatewayClassGVR, gatewayClass("nginx", "example.com/other-controller"))
	create(t, dc, client.GatewayGVR, gateway("default", "eg-gw", "eg"))
	create(t, dc, client.GatewayGVR, gateway("default", "other-gw", "nginx"))
	create(t, dc, client.HTTPRouteGVR, httpRoute("default", "shop", "eg-gw", []string{"shop.example.com"}))
	create(t, dc, client.HTTPRouteGVR, httpRoute("default", "other", "other-gw", []string{"other.example.com"}))

	discovery := &httpRouteDiscovery{k8s: k8sClient}

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		targets, err := discovery.DiscoverTargets(context.Background())
		assert.NoError(c, err)
		require.Len(c, targets, 1)
		target := targets[0]
		assert.Equal(c, EnvoyGatewayHttpRouteTargetType, target.TargetType)
		assert.Equal(c, "shop", target.Label)
		assert.Equal(c, []string{"shop"}, target.Attributes["k8s.envoy-gateway.http-route"])
		assert.Equal(c, []string{"shop.example.com"}, target.Attributes["k8s.envoy-gateway.http-route.hostname"])
		assert.Equal(c, []string{"eg-gw"}, target.Attributes["k8s.envoy-gateway.gateway"])
		assert.Equal(c, []string{"eg"}, target.Attributes["k8s.envoy-gateway.gatewayclass"])
		assert.Equal(c, []string{"test-cluster"}, target.Attributes["k8s.cluster-name"])
	}, 3*time.Second, 50*time.Millisecond)
}

// --- Action lifecycle --------------------------------------------------------

func newDelayRequest(executionId uuid.UUID) action_kit_api.PrepareActionRequestBody {
	return action_kit_api.PrepareActionRequestBody{
		ExecutionId: executionId,
		Config:      map[string]any{"duration": float64(30000), "percentage": float64(50), "delay": float64(5000)},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace":                {"default"},
				"k8s.envoy-gateway.http-route": {"shop"},
			},
		}),
	}
}

func Test_action_lifecycle_createsAndDeletesPolicy(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sClient, dc := getTestClient(stopCh)

	action := NewDelayAction(k8sClient).(*backendTrafficPolicyAction)
	executionId := uuid.New()
	req := newDelayRequest(executionId)

	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, req)
	require.NoError(t, err)
	assert.Equal(t, "steadybit-delay-"+executionId.String(), state.PolicyName)

	_, err = action.Start(context.Background(), &state)
	require.NoError(t, err)

	created, err := dc.Resource(client.BackendTrafficPolicyGVR).Namespace("default").Get(context.Background(), state.PolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, managedByValue, created.GetLabels()[managedByLabelKey])

	_, err = action.Stop(context.Background(), &state)
	require.NoError(t, err)

	_, err = dc.Resource(client.BackendTrafficPolicyGVR).Namespace("default").Get(context.Background(), state.PolicyName, metav1.GetOptions{})
	assert.Error(t, err, "policy should be deleted on stop")
}

func Test_action_prepare_failsOnConflictingPolicy(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sClient, dc := getTestClient(stopCh)

	existing := policyTargeting("pre-existing", "shop", "")
	existing.SetNamespace("default")
	existing.Object["apiVersion"] = btpAPIVersion
	existing.Object["kind"] = btpKind
	create(t, dc, client.BackendTrafficPolicyGVR, &existing)

	action := NewDelayAction(k8sClient).(*backendTrafficPolicyAction)
	req := newDelayRequest(uuid.New())
	state := action.NewEmptyState()

	_, err := action.Prepare(context.Background(), &state, req)
	assert.Error(t, err, "prepare should fail when a policy already targets the route")
}
