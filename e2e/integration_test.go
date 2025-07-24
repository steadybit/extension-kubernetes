// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/steadybit/extension-kit/extutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	validateAdvice "github.com/steadybit/advice-kit/go/advice_kit_test/validate"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_test/validate"
	"github.com/steadybit/extension-kubernetes/v2/extcluster"
	"github.com/steadybit/extension-kubernetes/v2/extcontainer"
	"github.com/steadybit/extension-kubernetes/v2/extdeployment"
	"github.com/steadybit/extension-kubernetes/v2/extingress"
	"github.com/steadybit/extension-kubernetes/v2/extnode"
	"github.com/steadybit/extension-kubernetes/v2/extpod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	acorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/utils/strings/slices"
)

var testCases = []e2e.WithMinikubeTestCase{
	{
		Name: "validate discovery",
		Test: validateDiscovery,
	},
	{
		Name: "validate advice",
		Test: validateAdviceDiscovery,
	},
	{
		Name: "discovery",
		Test: testDiscovery,
	},
	{
		Name: "checkRolloutTwice",
		Test: testCheckRolloutTwice,
	},
	{
		Name: "checkRolloutReady",
		Test: testCheckRolloutReady,
	},
	{
		Name: "deletePod",
		Test: testDeletePod,
	},
	{
		Name: "drainNode",
		Test: testDrainNode,
	},
	{
		Name: "taintNode",
		Test: testTaintNode,
	},
	{
		Name: "scaleDeployment",
		Test: testScaleDeployment,
	},
	{
		Name: "causeCrashLoop",
		Test: testCauseCrashLoop,
	},
	{
		Name: "setImage",
		Test: testSetImage,
	},
	{
		Name: "haproxyDelayTraffic",
		Test: testHAProxyDelayTraffic,
	},
	{
		Name: "haproxyBlockTraffic",
		Test: testHAProxyBlockTraffic,
	},
	{
		Name: "nginxIngressDiscovery",
		Test: testNginxIngressDiscovery,
	},
	{
		Name: "nginxBlockTraffic",
		Test: testNginxBlockTraffic,
	},
	{
		Name: "nginxDelayTraffic",
		Test: testNginxDelayTraffic,
	},
	{
		Name: "nginxMultipleControllers",
		Test: testNginxMultipleControllers,
	},
}

func TestWithMinikube(t *testing.T) {
	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-kubernetes",
		Port: 8088,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", "kubernetes.clusterName=e2e-cluster",
				"--set", "discovery.attributes.excludes.container={k8s.label.*}",
				"--set", "discovery.refreshThrottle=1",
				"--set", "logging.level=INFO",
			}
		},
	}

	e2e.WithDefaultMinikube(t, &extFactory, testCases)
}

func TestWithMinikubeViaRole(t *testing.T) {
	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-kubernetes",
		Port: 8088,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", "kubernetes.clusterName=e2e-cluster",
				"--set", "discovery.attributes.excludes.container={k8s.label.*}",
				"--set", "discovery.refreshThrottle=1",
				"--set", "logging.level=debug",
				"--set", "role.create=true",
				"--set", "kubernetes.namespaceFilter=default",
				"--set", "roleBinding.create=true",
				"--set", "clusterRole.create=false",
				"--set", "clusterRoleBinding.create=false",
				"--namespace", "default",
			}
		},
	}
	// add env var to use role binding to configure the tests
	t.Setenv("USE_K8S_ROLE_BINDING", "true")

	e2e.WithDefaultMinikube(t, &extFactory, testCases)
}

func validateDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, validate.ValidateEndpointReferences("/", e.Client))
}

func validateAdviceDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, validateAdvice.ValidateEndpointReferences("/", e.Client))
}

func testCheckRolloutReady(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testCheckRolloutReady")

	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx-check-rollout-ready")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()

	tests := []struct {
		name            string
		wantedCompleted bool
	}{
		{
			name:            "should check status ok",
			wantedCompleted: true,
		},
		{
			name:            "should check status not completed",
			wantedCompleted: false,
		},
	}

	require.NoError(t, err)

	for _, tt := range tests {

		config := struct {
			Duration int `json:"duration"`
		}{
			Duration: 15000,
		}

		t.Run(tt.name, func(t *testing.T) {
			if tt.wantedCompleted {
				exec, err := m.PodExec(e.Pod, "extension", "kubectl", "rollout", "restart", "deployment/nginx-check-rollout-ready")
				require.NoError(t, err)
				log.Info().Msgf("kubectl rollout restart deployment/nginx-check-rollout-ready: %s", exec)
			} else {
				exec, err := m.PodExec(e.Pod, "extension", "kubectl", "rollout", "restart", "deployment/nginx-check-rollout-ready")
				require.NoError(t, err)
				log.Info().Msgf("kubectl rollout restart deployment/nginx-check-rollout-ready: %s", exec)
				exec, err = m.PodExec(e.Pod, "extension", "kubectl", "rollout", "pause", "deployment/nginx-check-rollout-ready")
				require.NoError(t, err)
				log.Info().Msgf("kubectl rollout pause deployment/nginx-check-rollout-ready: %s", exec)
			}

			target := action_kit_api.Target{
				Name: "test",
				Attributes: map[string][]string{
					"k8s.cluster-name": {"e2e-cluster"},
					"k8s.namespace":    {"default"},
					"k8s.deployment":   {"nginx-check-rollout-ready"},
				},
			}
			action, err := e.RunAction(extdeployment.RolloutStatusActionId, &target, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			err = action.Wait()
			if tt.wantedCompleted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}

}

func testCheckRolloutTwice(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testCheckRolloutTwice")

	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx-check-rollout-twice")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()
	// Update the deployment to add a readiness probe
	cmdOut, cmdErr := m.PodExec(e.Pod, "extension", "kubectl", "patch", "deployment", "nginx-check-rollout-twice", "-n", "default", "--type", "json", "-p",
		`[{
        "op": "add",
        "path": "/spec/template/spec/containers/0/readinessProbe",
        "value": {
            "initialDelaySeconds": 5,
            "periodSeconds": 3,
            "failureThreshold": 3,
            "timeoutSeconds": 1,
            "successThreshold": 1,
            "tcpSocket": {
                "port": 80
            }
        }
    }]`)
	if cmdErr != nil {
		log.Error().Msgf("Failed to patch deployment: %s, %v", cmdOut, cmdErr)
		require.NoError(t, cmdErr, "failed to patch deployment")
	}
	// Wait for rollout to complete after patching
	waitCmd, waitErr := m.PodExec(e.Pod, "extension", "kubectl", "rollout", "status", "deployment/nginx-check-rollout-twice", "-n", "default", "--timeout=30s")
	if waitErr != nil {
		log.Error().Msgf("Failed to wait for rollout completion: %s, %v", waitCmd, waitErr)
		require.NoError(t, waitErr, "failed to wait for rollout completion")
	}
	log.Info().Msg("Deployment patched and rollout completed")

	tests := []struct {
		name      string
		firstWait bool
	}{
		{
			name:      "should rollout twice, second time should succeed",
			firstWait: true,
		},
		{
			name:      "should rollout twice, second time should fail",
			firstWait: false,
		},
	}

	require.NoError(t, err)

	for _, tt := range tests {

		config := struct {
			Duration    int  `json:"duration"`
			Wait        bool `json:"wait"`
			CheckBefore bool `json:"checkBefore"`
		}{
			Duration:    5000,
			Wait:        tt.firstWait,
			CheckBefore: true,
		}

		t.Run(tt.name, func(t *testing.T) {
			target := action_kit_api.Target{
				Name: "test",
				Attributes: map[string][]string{
					"k8s.cluster-name": {"e2e-cluster"},
					"k8s.namespace":    {"default"},
					"k8s.deployment":   {"nginx-check-rollout-twice"},
				},
			}
			action, err := e.RunAction(extdeployment.RolloutRestartActionId, &target, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			err = action.Wait()
			require.NoError(t, err)
			secoundTimeAction, secondErr := e.RunAction(extdeployment.RolloutRestartActionId, &target, config, nil)
			defer func() { _ = secoundTimeAction.Cancel() }()
			if tt.firstWait {
				require.NoError(t, secondErr)
				require.NoError(t, secoundTimeAction.Wait())
			} else {
				require.Error(t, secondErr)
				assert.Contains(t, secondErr.Error(), "Cannot start rollout restart: there is already an ongoing rollout for this deployment")
			}
		})
	}

}

func isUsingRoleBinding() bool {
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if pair[0] == "USE_K8S_ROLE_BINDING" {
			return pair[1] == "true"
		}
	}
	return false
}

func testDiscovery(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testDiscovery")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()

	target, err := e2e.PollForTarget(ctx, e, extdeployment.DeploymentTargetType, func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "k8s.deployment", "nginx") && e2e.HasAttribute(target, "k8s.pod.name", nginx.Pods[0].Name) && e2e.HasAttribute(target, "k8s.pod.name", nginx.Pods[1].Name)
	})

	require.NoError(t, err)
	assert.Equal(t, target.TargetType, extdeployment.DeploymentTargetType)
	assert.Equal(t, target.Attributes["k8s.namespace"][0], "default")
	assert.Equal(t, target.Attributes["k8s.deployment"][0], "nginx")
	assert.Equal(t, target.Attributes["k8s.workload-type"][0], "deployment")
	assert.Equal(t, target.Attributes["k8s.workload-owner"][0], "nginx")
	assert.Equal(t, target.Attributes["k8s.deployment"][0], "nginx")
	assert.Equal(t, target.Attributes["k8s.deployment.label.app"][0], "nginx")
	assert.Equal(t, target.Attributes["k8s.cluster-name"][0], "e2e-cluster")
	assert.Contains(t, target.Attributes["k8s.pod.name"], nginx.Pods[0].Name)
	assert.Contains(t, target.Attributes["k8s.pod.name"], nginx.Pods[1].Name)
	assert.Equal(t, target.Attributes["k8s.distribution"][0], "kubernetes")

	enrichmentData, err := e2e.PollForEnrichmentData(ctx, e, extcontainer.KubernetesContainerEnrichmentDataType, func(enrichmentData discovery_kit_api.EnrichmentData) bool {
		return e2e.ContainsAttribute(enrichmentData.Attributes, "k8s.container.name", "nginx")
	})

	require.NoError(t, err)
	assert.Equal(t, enrichmentData.EnrichmentDataType, extcontainer.KubernetesContainerEnrichmentDataType)
	assert.Equal(t, enrichmentData.Attributes["k8s.container.name"][0], "nginx")
	assert.Equal(t, enrichmentData.Attributes["k8s.container.image"][0], "nginx:stable-alpine")
	assert.Equal(t, enrichmentData.Attributes["k8s.pod.label.app"][0], "nginx")
	assert.Equal(t, enrichmentData.Attributes["k8s.namespace"][0], "default")
	assert.Equal(t, enrichmentData.Attributes["k8s.node.name"][0], "e2e-docker")
	assert.NotContains(t, enrichmentData.Attributes, "k8s.label.app")

	podNames := make([]string, 0, len(nginx.Pods))
	for _, pod := range nginx.Pods {
		podNames = append(podNames, pod.Name)
	}
	assert.Contains(t, podNames, enrichmentData.Attributes["k8s.pod.name"][0])

	target, err = e2e.PollForTarget(ctx, e, extcluster.ClusterTargetType, func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "k8s.cluster-name", "e2e-cluster")
	})
	require.NoError(t, err)
	assert.Equal(t, target.TargetType, extcluster.ClusterTargetType)

	target, err = e2e.PollForTarget(ctx, e, extpod.PodTargetType, func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "k8s.deployment", "nginx")
	})
	require.NoError(t, err)
	assert.Equal(t, target.TargetType, extpod.PodTargetType)

	if !isUsingRoleBinding() {
		target, err = e2e.PollForTarget(ctx, e, extnode.NodeTargetType, func(target discovery_kit_api.Target) bool {
			return true
		})
		require.NoError(t, err)
		assert.Equal(t, target.TargetType, extnode.NodeTargetType)
		assert.Equal(t, "e2e-docker", target.Attributes["host.hostname"][0])
	}

	// Initialize HAProxy and test resources
	if !isUsingRoleBinding() {

		_, testAppName, _, nginxDeployment, appService, appIngress := initHAProxy(t, m, e, ctx, "haproxy-controller")
		defer func() { _ = m.DeleteDeployment(nginxDeployment) }()
		defer func() { _ = m.DeleteService(appService) }()
		defer func() { _ = m.DeleteIngress(appIngress) }()
		defer func() {
			cleanupHAProxy(m, "haproxy-controller")
		}()

		haproxy, err := e2e.PollForTarget(ctx, e, extingress.HAProxyIngressTargetType, func(target discovery_kit_api.Target) bool {
			return e2e.HasAttribute(target, "k8s.ingress", testAppName)
		})
		require.NoError(t, err)
		assert.Equal(t, haproxy.TargetType, extingress.HAProxyIngressTargetType)
		assert.Equal(t, haproxy.Attributes["k8s.ingress"][0], testAppName)
		assert.Equal(t, haproxy.Attributes["k8s.ingress.controller"][0], "haproxy.org/ingress-controller/haproxy")
		assert.Equal(t, haproxy.Attributes["k8s.ingress.class"][0], "haproxy")
	}
}

func testDeletePod(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testDeletePod")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//Start Deployment with 2 pods
	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx-test-delete-pod")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()
	podName1 := nginx.Pods[0].Name
	podName2 := nginx.Pods[1].Name
	log.Info().Msgf("Pods before Attack: podName1: %v, podName2: %v", podName1, podName2)

	//Delete both pods
	_, err = e.RunAction(extpod.DeletePodActionId, &action_kit_api.Target{
		Name:       podName1,
		Attributes: map[string][]string{"k8s.pod.name": {podName1}, "k8s.namespace": {"default"}},
	}, nil, nil)
	require.NoError(t, err)
	_, err = e.RunAction(extpod.DeletePodActionId, &action_kit_api.Target{
		Name:       podName2,
		Attributes: map[string][]string{"k8s.pod.name": {podName2}, "k8s.namespace": {"default"}},
	}, nil, nil)
	require.NoError(t, err)

	//Check if new pods are coming up
	_, err = e2e.PollForTarget(ctx, e, extdeployment.DeploymentTargetType, func(target discovery_kit_api.Target) bool {
		log.Debug().Msgf("pod: %v", target.Attributes["k8s.pod.name"])
		return e2e.HasAttribute(target, "k8s.deployment", "nginx-test-delete-pod") &&
			len(target.Attributes["k8s.pod.name"]) == 2 &&
			podName1 != target.Attributes["k8s.pod.name"][0] &&
			podName2 != target.Attributes["k8s.pod.name"][0]
	})
	require.NoError(t, err)
}

func testDrainNode(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	if isUsingRoleBinding() {
		log.Info().Msg("Skipping testDrainNode because it is using role binding, and is therefore not supported")
		return
	}

	log.Info().Msg("Starting testDrainNode")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	//Start Deployment with 2 pods
	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx-test-drain")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()
	podName1 := nginx.Pods[0].Name
	podName2 := nginx.Pods[1].Name
	assert.Len(t, nginx.Pods, 2)
	log.Info().Msgf("Pods before Attack: podName1: %v, podName2: %v", podName1, podName2)

	//Check if node discovery is working and listing both pods
	nodeTarget, err := e2e.PollForTarget(ctx, e, extnode.NodeTargetType, func(target discovery_kit_api.Target) bool {
		return slices.Contains(target.Attributes["k8s.pod.name"], podName1) && slices.Contains(target.Attributes["k8s.pod.name"], podName2)
	})
	require.NoError(t, err)

	//Drain node
	config := struct {
		Duration int `json:"duration"`
	}{
		Duration: 10000,
	}
	_, err = e.RunAction(extnode.DrainNodeActionId, &action_kit_api.Target{
		Name: nodeTarget.Id,
		Attributes: map[string][]string{
			"host.hostname": nodeTarget.Attributes["host.hostname"],
		},
	}, config, nil)
	require.NoError(t, err)

	// pods are removed
	_, err = e2e.PollForTarget(ctx, e, extnode.NodeTargetType, func(target discovery_kit_api.Target) bool {
		return !slices.Contains(target.Attributes["k8s.pod.name"], podName1) && !slices.Contains(target.Attributes["k8s.pod.name"], podName2)
	})
	require.NoError(t, err)
	log.Info().Msgf("pods are removed")

	// pods are rescheduled after attack
	_, err = e2e.PollForTarget(ctx, e, extnode.NodeTargetType, func(target discovery_kit_api.Target) bool {
		for _, pod := range target.Attributes["k8s.pod.name"] {
			if strings.HasPrefix(pod, "nginx-test-drain-") {
				return true
			}
		}
		return false
	})
	log.Info().Msgf("pods are rescheduled")
	require.NoError(t, err)
}

func testTaintNode(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	if isUsingRoleBinding() {
		log.Info().Msg("Skipping testDrainNode because it is using role binding, and is therefore not supported")
		return
	}
	log.Info().Msg("Starting testTaintNode")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	//Start Deployment with 2 pods
	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx-test-taint")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()
	podName1 := nginx.Pods[0].Name
	podName2 := nginx.Pods[1].Name
	assert.Len(t, nginx.Pods, 2)
	log.Info().Msgf("Pods before Attack: podName1: %v, podName2: %v", podName1, podName2)

	//Check if node discovery is working and listing both pods
	nodeTarget, err := e2e.PollForTarget(ctx, e, extnode.NodeTargetType, func(target discovery_kit_api.Target) bool {
		return slices.Contains(target.Attributes["k8s.pod.name"], podName1) && slices.Contains(target.Attributes["k8s.pod.name"], podName2)
	})
	require.NoError(t, err)

	//Taint node
	config := struct {
		Duration int    `json:"duration"`
		Key      string `json:"key"`
		Value    string `json:"value"`
		Effect   string `json:"effect"`
	}{
		Duration: 20_000,
		Key:      "allowed",
		Value:    "nothing",
		Effect:   "NoSchedule",
	}
	_, err = e.RunAction(extnode.DrainNodeActionId, &action_kit_api.Target{
		Name: nodeTarget.Id,
		Attributes: map[string][]string{
			"host.hostname": nodeTarget.Attributes["host.hostname"],
		},
	}, config, nil)
	require.NoError(t, err)
	attackStarted := time.Now()

	//Delete both pods
	_, err = e.RunAction(extpod.DeletePodActionId, &action_kit_api.Target{
		Name:       podName1,
		Attributes: map[string][]string{"k8s.pod.name": {podName1}, "k8s.namespace": {"default"}},
	}, nil, nil)
	require.NoError(t, err)
	_, err = e.RunAction(extpod.DeletePodActionId, &action_kit_api.Target{
		Name:       podName1,
		Attributes: map[string][]string{"k8s.pod.name": {podName2}, "k8s.namespace": {"default"}},
	}, nil, nil)
	require.NoError(t, err)

	// pods are removed and do not come back as long as the node is tainted
	_, err = e2e.PollForTarget(ctx, e, extnode.NodeTargetType, func(target discovery_kit_api.Target) bool {
		containsNginxPod := false
		for _, pod := range target.Attributes["k8s.pod.name"] {
			if strings.HasPrefix(pod, "nginx-test-taint-") {
				containsNginxPod = true
			}
		}
		return (time.Since(attackStarted) > 10*time.Second) && !containsNginxPod
	})
	require.NoError(t, err)
	log.Info().Msgf("pods didn't come back within 10 seconds, node seems to be tainted")

	// pods are rescheduled after attack
	_, err = e2e.PollForTarget(ctx, e, extnode.NodeTargetType, func(target discovery_kit_api.Target) bool {
		for _, pod := range target.Attributes["k8s.pod.name"] {
			if strings.HasPrefix(pod, "nginx-test-taint-") {
				return true
			}
		}
		return false
	})
	log.Info().Msgf("pods are rescheduled")
	require.NoError(t, err)
}

func testScaleDeployment(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testScaleDeployment")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	//Start Deployment with 2 pods
	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx-test-scale")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()
	assert.Len(t, nginx.Pods, 2)

	var distinctPodNames = make(map[string]string)
	//Check if node discovery is working and listing 2 pods
	nodeTarget, err := e2e.PollForTarget(ctx, e, extpod.PodTargetType, func(target discovery_kit_api.Target) bool {
		for _, pod := range target.Attributes["k8s.pod.name"] {
			if strings.HasPrefix(pod, "nginx-test-scale-") {
				distinctPodNames[pod] = pod
			}
		}
		return len(distinctPodNames) == 2
	})
	require.NoError(t, err)

	//scale deployment
	config := struct {
		Duration     int `json:"duration"`
		ReplicaCount int `json:"replicaCount"`
	}{
		Duration:     10000,
		ReplicaCount: 3,
	}
	_, err = e.RunAction(extdeployment.ScaleDeploymentActionId, &action_kit_api.Target{
		Name: nodeTarget.Id,
		Attributes: map[string][]string{
			"k8s.namespace":  {"default"},
			"k8s.deployment": {"nginx-test-scale"},
		},
	}, config, nil)
	require.NoError(t, err)

	// pods are upscaled
	distinctPodNames = make(map[string]string)
	_, err = e2e.PollForTarget(ctx, e, extpod.PodTargetType, func(target discovery_kit_api.Target) bool {
		for _, pod := range target.Attributes["k8s.pod.name"] {
			if strings.HasPrefix(pod, "nginx-test-scale-") {
				distinctPodNames[pod] = pod
			}
		}
		return len(distinctPodNames) == 3
	})
	require.NoError(t, err)
	log.Info().Msgf("pods are scaled to 3")

	// pod scale is reverted to 2 after attack
	distinctPodNames = make(map[string]string)
	_, err = e2e.PollForTarget(ctx, e, extpod.PodTargetType, func(target discovery_kit_api.Target) bool {
		for _, pod := range target.Attributes["k8s.pod.name"] {
			if strings.HasPrefix(pod, "nginx-test-scale-") {
				distinctPodNames[pod] = pod
			}
		}
		return len(distinctPodNames) == 2
	})
	require.NoError(t, err)
	log.Info().Msgf("pod replica count is back to 2")
}

func testCauseCrashLoop(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testCauseCrashLoop")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	//Start Deployment with 2 pods
	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx-test-crashloop")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()
	require.Len(t, nginx.Pods, 2)
	firstPod := nginx.Pods[0]
	require.Equal(t, int32(0), firstPod.Status.ContainerStatuses[0].RestartCount)

	target, err := e2e.PollForTarget(ctx, e, extpod.PodTargetType, func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "k8s.pod.name", firstPod.Name)
	})
	require.NoError(t, err)

	//CrashLoopPod
	config := struct {
		Duration int `json:"duration"`
	}{
		Duration: 30_000,
	}
	action, err := e.RunAction(extpod.CrashLoopActionId, &action_kit_api.Target{
		Name: target.Id,
		Attributes: map[string][]string{
			"k8s.namespace": {"default"},
			"k8s.pod.name":  {firstPod.Name},
		},
	}, config, nil)
	defer func() { _ = action.Cancel() }()
	require.NoError(t, err)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		p, err := m.GetPod(firstPod.GetObjectMeta())
		require.NoError(t, err)
		assert.GreaterOrEqual(collect, p.Status.ContainerStatuses[0].RestartCount, int32(2))
	}, 20*time.Second, 1*time.Second, "pod should be restarted at least twice")
}

func testSetImage(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testSetImage")
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Start Deployment with 2 pods
	nginx := e2e.NginxDeployment{Minikube: m}
	err := nginx.Deploy("nginx-test-set-image")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginx.Delete() }()
	assert.Len(t, nginx.Pods, 2)

	var distinctPodNames = make(map[string]string)

	// Check if node discovery is working and listing 2 pods
	target, err := e2e.PollForTarget(ctx, e, extpod.PodTargetType, func(target discovery_kit_api.Target) bool {
		for _, pod := range target.Attributes["k8s.pod.name"] {
			if strings.HasPrefix(pod, "nginx-test-set-image-") {
				distinctPodNames[pod] = pod
			}
		}
		return len(distinctPodNames) == 2
	})

	require.NoError(t, err)

	// Set image
	config := struct {
		Duration      int    `json:"duration"`
		Image         string `json:"image"`
		ContainerName string `json:"container_name"`
	}{
		Duration:      120000,
		Image:         "httpd:alpine",
		ContainerName: "nginx",
	}

	action, err := e.RunAction(
		extdeployment.SetImageActionId,
		&action_kit_api.Target{
			Name: target.Id,
			Attributes: map[string][]string{
				"k8s.namespace":      {"default"},
				"k8s.deployment":     {"nginx-test-set-image"},
				"k8s.container.name": {"nginx"},
			},
		}, config, nil)

	defer func() { _ = action.Cancel() }()

	require.NoError(t, err)

	newDistinctPodNames := make(map[string]string)
	httpdPodsCount := 0

	// Verify creation of new pods
	_, err = e2e.PollForTarget(ctx, e, extpod.PodTargetType, func(target discovery_kit_api.Target) bool {
		for _, pod := range target.Attributes["k8s.pod.name"] {
			if strings.HasPrefix(pod, "nginx-test-set-image-") {
				_, ok := distinctPodNames[pod]

				if !ok {
					newDistinctPodNames[pod] = pod
				}
			}
		}

		return len(newDistinctPodNames) == 2
	})

	require.NoError(t, err)

	// Verify new pods have the new image
	for _, pod := range newDistinctPodNames {
		podMeta := metav1.ObjectMeta{
			Name:      pod,
			Namespace: "default",
		}

		p, err := m.GetPod(podMeta.GetObjectMeta())
		require.NoError(t, err)

		containers := p.Spec.Containers

		for _, container := range containers {
			if container.Name == "nginx" && container.Image == "httpd:alpine" {
				httpdPodsCount++
			}
		}
	}

	assert.Equal(t, httpdPodsCount, 2)
}

func testHAProxyDelayTraffic(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	if isUsingRoleBinding() {
		log.Info().Msg("Skipping testHAProxyDelayTraffic because it is using role binding, and is therefore not supported")
		return
	}
	log.Info().Msg("Starting testHAProxyDelayTraffic")
	const haProxyControllerNamespace = "haproxy-controller"
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Initialize HAProxy and test resources
	haProxyService, testAppName, ingressTarget, nginxDeployment, appService, appIngress := initHAProxy(t, m, e, ctx, haProxyControllerNamespace)
	defer func() { _ = m.DeleteDeployment(nginxDeployment) }()
	defer func() { _ = m.DeleteService(appService) }()
	defer func() { _ = m.DeleteIngress(appIngress) }()
	defer func() {
		cleanupHAProxy(m, haProxyControllerNamespace)
	}()

	// Measure baseline latency
	baselineLatency, err := measureRequestLatency(m, haProxyService, testAppName+".local")
	require.NoError(t, err)
	log.Info().Msgf("Baseline latency: %v", baselineLatency)

	// Define delay parameters
	delayMs := 500
	tests := []struct {
		name                 string
		responseDelay        int
		conditionPathPattern string
		conditionHttpMethod  string
		conditionHttpHeader  []interface{}
		requestPath          string
		requestHeaders       map[string]string
		requestMethod        string
		wantedDelay          bool
	}{
		{
			name:                 "should delay traffic for the specified path",
			requestPath:          "/",
			conditionPathPattern: "/",
			responseDelay:        delayMs,
			wantedDelay:          true,
		},
		{
			name:                 "should not delay traffic for mismatched path",
			requestPath:          "/",
			conditionPathPattern: "/api",
			responseDelay:        delayMs,
			wantedDelay:          false,
		},
		{
			name:                "should delay traffic for specified HTTP method",
			requestPath:         "/",
			responseDelay:       delayMs,
			conditionHttpMethod: "GET",
			wantedDelay:         true,
		},
		{
			name:                "should not delay traffic for mismatched HTTP method",
			requestPath:         "/",
			conditionHttpMethod: "POST",
			wantedDelay:         false,
		},
		{
			name:        "should delay traffic for specified HTTP header",
			requestPath: "/",
			requestHeaders: map[string]string{
				"User-Agent": "Mozilla/5.0",
			},
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
			},
			responseDelay: delayMs,
			wantedDelay:   true,
		},
		{
			name:        "should not delay traffic for mismatched HTTP header",
			requestPath: "/",
			requestHeaders: map[string]string{
				"User-Agent": "Chrome/90.0",
			},
			responseDelay: delayMs,
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
			},
			wantedDelay: false,
		},
		{
			name:          "should delay traffic for combined conditions (all match)",
			requestPath:   "/",
			requestMethod: "GET",
			requestHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			responseDelay:        delayMs,
			conditionPathPattern: "/",
			conditionHttpMethod:  "GET",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "Content-Type", "value": "application/json"},
			},
			wantedDelay: true,
		},
		{
			name:          "should not delay traffic for combined conditions (one mismatch)",
			requestPath:   "/",
			requestMethod: "GET", // Mismatch - config requires POST
			requestHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			responseDelay:        delayMs,
			conditionPathPattern: ".*",
			conditionHttpMethod:  "POST",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "Content-Type", "value": "application/json"},
			},
			wantedDelay: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply delay traffic action
			config := struct {
				Duration             int           `json:"duration"`
				ResponseDelay        int           `json:"responseDelay"`
				ConditionPathPattern string        `json:"conditionPathPattern,omitempty"`
				ConditionHttpMethod  string        `json:"conditionHttpMethod,omitempty"`
				ConditionHttpHeader  []interface{} `json:"conditionHttpHeader,omitempty"`
			}{
				Duration:             30000,
				ResponseDelay:        tt.responseDelay,
				ConditionPathPattern: tt.conditionPathPattern,
				ConditionHttpMethod:  tt.conditionHttpMethod,
				ConditionHttpHeader:  tt.conditionHttpHeader,
			}

			log.Info().Msgf("Applying delay of %dms to path %s", tt.responseDelay, tt.conditionPathPattern)
			action, err := e.RunAction(extingress.HAProxyDelayTrafficActionId, ingressTarget, config, nil)
			require.NoError(t, err)
			defer func() { _ = action.Cancel() }()

			// Measure latency during delay
			time.Sleep(5 * time.Second) // Give HAProxy time to reconfigure

			// Use the correct method and headers for the test
			var delayedLatency time.Duration
			if tt.requestMethod != "" || len(tt.requestHeaders) > 0 {
				delayedLatency, err = measureRequestLatencyWithOptions(
					m,
					haProxyService,
					testAppName+".local",
					tt.requestPath,
					tt.requestMethod,
					tt.requestHeaders,
				)
			} else {
				delayedLatency, err = measureRequestLatency(m, haProxyService, testAppName+".local")
			}
			require.NoError(t, err)
			log.Info().Msgf("Latency during delay test: %v", delayedLatency)

			// Verify delay
			if tt.wantedDelay {
				// Check that delay is applied (with some tolerance)
				minExpectedLatency := baselineLatency + time.Duration(delayMs-50)*time.Millisecond  // -50ms tolerance
				maxExpectedLatency := baselineLatency + time.Duration(delayMs+200)*time.Millisecond // +200ms tolerance for overhead
				assert.GreaterOrEqual(t, delayedLatency, minExpectedLatency, "Latency should increase by approximately the configured delay")
				assert.LessOrEqual(t, delayedLatency, maxExpectedLatency, "Latency should not be much higher than expected")
			} else {
				// Latency shouldn't change significantly
				maxExpectedLatency := baselineLatency + 100*time.Millisecond // Allow for some small variance
				assert.LessOrEqual(t, delayedLatency, maxExpectedLatency, "Latency should not increase significantly")
			}

			// Cancel the action
			require.NoError(t, action.Cancel())

			// Measure latency after cancellation
			time.Sleep(5 * time.Second) // Give HAProxy time to reconfigure
			var afterLatency time.Duration
			if tt.requestMethod != "" || len(tt.requestHeaders) > 0 {
				afterLatency, err = measureRequestLatencyWithOptions(
					m,
					haProxyService,
					testAppName+".local",
					tt.requestPath,
					tt.requestMethod,
					tt.requestHeaders,
				)
			} else {
				afterLatency, err = measureRequestLatency(m, haProxyService, testAppName+".local")
			}
			require.NoError(t, err)
			log.Info().Msgf("Latency after cancellation: %v", afterLatency)

			// Verify latency returned to normal
			maxExpectedAfterLatency := baselineLatency + 100*time.Millisecond
			assert.LessOrEqual(t, afterLatency, maxExpectedAfterLatency, "Latency should return to normal after cancellation")
		})
	}
}

func cleanupHAProxy(m *e2e.Minikube, haProxyControllerNamespace string) {
	_ = exec.Command("helm", "uninstall", "haproxy-ingress", "--namespace", "--kube-context", m.Profile, haProxyControllerNamespace).Run()
	// check if clusterrole is still present
	if checkClusterRole(m, "haproxy-ingress-kubernetes-ingress") {
		deleteClusterRole(m, "haproxy-ingress-kubernetes-ingress")
	}
}

func testHAProxyBlockTraffic(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	if isUsingRoleBinding() {
		log.Info().Msg("Skipping testHAProxyBlockTraffic because it is using role binding, and is therefore not supported")
		return
	}
	log.Info().Msg("Starting testHAProxyBlockTraffic")
	const haProxyControllerNamespace = "haproxy-controller"
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Initialize HAProxy and test resources
	haProxyService, _, ingressTarget, nginxDeployment, appService, appIngress := initHAProxy(t, m, e, ctx, haProxyControllerNamespace)
	defer func() { _ = m.DeleteDeployment(nginxDeployment) }()
	defer func() { _ = m.DeleteService(appService) }()
	defer func() { _ = m.DeleteIngress(appIngress) }()
	defer func() {
		cleanupHAProxy(m, haProxyControllerNamespace)
	}()

	// Service should be reachable via haproxy service
	err := checkStatusCode(t, m, haProxyService, "/", 200)
	require.NoError(t, err)
	log.Info().Msgf("App is available")

	// Define delay parameters
	tests := []struct {
		name                 string
		testPath             string
		responseStatusCode   int
		conditionPathPattern string
		conditionHttpMethod  string
		conditionHttpHeader  []interface{}
		requestHeaders       map[string]string
		wantedBlock          bool
	}{
		{
			name:                 "should block traffic for the specified path",
			testPath:             "/",
			conditionPathPattern: "/",
			responseStatusCode:   503,
			wantedBlock:          true,
		},
		{
			name:                 "should block traffic for a specific endpoint",
			testPath:             "/",
			conditionPathPattern: "/api",
			responseStatusCode:   200,
			wantedBlock:          false, // We're requesting /, so /api block shouldn't affect us
		},
		{
			name:                "should block traffic for a http method",
			testPath:            "/",
			conditionHttpMethod: "GET",
			responseStatusCode:  503,
			wantedBlock:         true,
		},
		{
			name:                "should not block traffic for a http method",
			testPath:            "/",
			conditionHttpMethod: "DELETE",
			responseStatusCode:  200,
			wantedBlock:         false, // We're requesting DELETE, so / block shouldn't affect us
		},
		{
			name:                 "should block traffic for a http method and path",
			testPath:             "/",
			conditionHttpMethod:  "GET",
			conditionPathPattern: "/",
			responseStatusCode:   501,
			wantedBlock:          true,
		},
		{
			name:                 "should not block traffic for a http method and path",
			testPath:             "/",
			conditionHttpMethod:  "DELETE",
			conditionPathPattern: "/",
			responseStatusCode:   200,
			wantedBlock:          false, // We're requesting DELETE, so / block shouldn't affect us
		},
		{
			name:     "should block traffic for a specific header",
			testPath: "/",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
			},
			requestHeaders:     map[string]string{"User-Agent": "Mozilla/5.0"},
			responseStatusCode: 501,
			wantedBlock:        true,
		},
		{
			name:                 "should block traffic with combined header and path conditions",
			testPath:             "/",
			conditionPathPattern: "/",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "Content-Type", "value": "application/json"},
			},
			requestHeaders:     map[string]string{"Content-Type": "application/json"},
			responseStatusCode: 451,
			wantedBlock:        true,
		},
		{
			name:     "should not block traffic when header doesn't match",
			testPath: "/",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "X-Test-Header", "value": "specific-value"},
			},
			requestHeaders:     map[string]string{"X-Test-Header": "other-value"},
			responseStatusCode: 200,
			wantedBlock:        false, // We're not sending X-Test-Header
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply delay traffic action
			config := struct {
				Duration             int           `json:"duration"`
				ResponseStatusCode   int           `json:"responseStatusCode"`
				ConditionPathPattern string        `json:"conditionPathPattern"`
				ConditionHttpMethod  string        `json:"conditionHttpMethod"`
				ConditionHttpHeader  []interface{} `json:"conditionHttpHeader"`
			}{
				Duration:             30000,
				ResponseStatusCode:   tt.responseStatusCode,
				ConditionPathPattern: tt.conditionPathPattern,
				ConditionHttpMethod:  tt.conditionHttpMethod,
				ConditionHttpHeader:  tt.conditionHttpHeader,
			}

			log.Info().Msgf("Applying block to path %s", tt.conditionPathPattern)
			action, err := e.RunAction(extingress.HAProxyBlockTrafficActionId, ingressTarget, config, nil)
			require.NoError(t, err)
			defer func() { _ = action.Cancel() }()

			time.Sleep(5 * time.Second) // Give HAProxy time to reconfigure

			// Verify block
			expectedStatusCode := tt.responseStatusCode
			if expectedStatusCode == 0 {
				if tt.wantedBlock {
					log.Error().Msgf("Expected status code not found for path %s", tt.testPath)
					t.Fail()
				} else {
					expectedStatusCode = 200
				}
			}

			// Use the combined checkStatusCode function with optional headers
			if tt.conditionHttpHeader != nil {
				err = checkStatusCode(t, m, haProxyService, tt.testPath, expectedStatusCode, tt.requestHeaders)
			} else {
				err = checkStatusCode(t, m, haProxyService, tt.testPath, expectedStatusCode)
			}

			require.NoError(t, err)
			if tt.wantedBlock {
				log.Info().Msgf("Path %s is blocked with status %d as expected", tt.testPath, expectedStatusCode)
			} else {
				log.Info().Msgf("Path %s is not blocked, received status 200 as expected", tt.testPath)
			}

			// Cancel the action
			require.NoError(t, action.Cancel())

			time.Sleep(5 * time.Second) // Give HAProxy time to reconfigure

			// Verify service is not blocked anymore
			err = checkStatusCode(t, m, haProxyService, "/", 200)
			require.NoError(t, err, "Service should not be blocked anymore")
		})
	}
}

func initHAProxy(t *testing.T, m *e2e.Minikube, e *e2e.Extension, ctx context.Context, haProxyControllerNamespace string) (*corev1.Service, string, *action_kit_api.Target, metav1.Object, metav1.Object, metav1.Object) {
	// Step 1: Deploy HAProxy Ingress Controller
	log.Info().Msg("Deploying HAProxy Ingress Controller")
	out, err := exec.Command("helm", "repo", "add", "haproxytech", "https://haproxytech.github.io/helm-charts").CombinedOutput()
	require.NoError(t, err, "Failed to add HAProxy Helm repo: %s", out)
	out, err = exec.Command("helm", "repo", "update").CombinedOutput()
	require.NoError(t, err, "Failed to update Helm repos: %s", out)
	out, err = exec.Command("helm", "upgrade", "--install", "haproxy-ingress", "haproxytech/kubernetes-ingress", "--create-namespace", "--namespace", haProxyControllerNamespace, "--kube-context", m.Profile, "--set", "controller.service.type=NodePort").CombinedOutput()
	require.NoError(t, err, "Failed to deploy HAProxy Ingress Controller: %s", out)

	// Wait for HAProxy ingress controller to be ready
	log.Info().Msg("Waiting for HAProxy Ingress Controller to be ready")
	err = waitForHAProxyIngressController(m, ctx, haProxyControllerNamespace)
	require.NoError(t, err)

	// Step 2: Create a test deployment with service
	log.Info().Msg("Creating test deployment")
	testAppName := "haproxy-delay-test"

	// Create deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName,
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": testAppName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": testAppName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:stable-alpine",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	nginxDeployment, _, err := m.CreateDeployment(deployment)
	require.NoError(t, err)

	service := acorev1.Service(testAppName, "default").
		WithLabels(map[string]string{
			"app": testAppName,
		}).
		WithSpec(acorev1.ServiceSpec().
			WithSelector(map[string]string{
				"app": testAppName,
			}).
			WithPorts(acorev1.ServicePort().
				WithPort(80).
				WithTargetPort(intstr.FromInt32(80)),
			),
		)

	appService, err := m.CreateService(service)
	require.NoError(t, err)

	// Step 3: Create ingress resource
	log.Info().Msg("Creating ingress resource")
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName,
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "haproxy",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pathTypePtr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: testAppName,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	appIngress, err := m.CreateIngress(ingress)
	require.NoError(t, err)

	// Get HAProxy Ingress target
	log.Info().Msg("Finding HAProxy Ingress target")
	var ingressTarget *action_kit_api.Target
	discoveryTarget, err := e2e.PollForTarget(ctx, e, extingress.HAProxyIngressTargetType, func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "k8s.ingress", testAppName)
	})
	require.NoError(t, err, "Failed to find HAProxy ingress target")
	ingressTarget = &action_kit_api.Target{
		Attributes: discoveryTarget.Attributes,
	}

	// Find HAProxy Service
	log.Info().Msg("Finding HAProxy Service")
	haProxyServiceName, err := findServiceNameInNamespace(m, haProxyControllerNamespace)
	require.NoError(t, err, "Failed to find HAProxy service")
	// Measure baseline latency
	log.Info().Msg("Measuring baseline latency")
	haProxyService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      haProxyServiceName,
			Namespace: haProxyControllerNamespace,
		},
	}

	return haProxyService, testAppName, ingressTarget, nginxDeployment, appService, appIngress
}

func waitForHAProxyIngressController(m *e2e.Minikube, ctx context.Context, namespace string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			pods, err := m.GetClient().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Error().Err(err).Msgf("Failed to check HAProxy controller status in namespace %s", namespace)
				continue
			}

			for _, pod := range pods.Items {
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if containerStatus.Ready && pod.Status.Phase == corev1.PodRunning {
						log.Info().Msgf("HAProxy controller is ready in namespace %s", namespace)
						return nil
					}
				}
			}
			log.Info().Msgf("Waiting for HAProxy controller in namespace %s to be ready...", namespace)
		}
	}
}

func findServiceNameInNamespace(m *e2e.Minikube, namespace string) (string, error) {
	services, err := m.GetClient().CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Failed to get service name")
		return "", err
	}
	if len(services.Items) == 0 {
		return "", fmt.Errorf("no services found in namespace %s", namespace)
	}
	return services.Items[0].Name, nil
}

func checkClusterRole(m *e2e.Minikube, name string) bool {
	list, err := m.GetClient().RbacV1().ClusterRoles().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list cluster roles")
		return false
	}
	for _, role := range list.Items {
		if role.Name == name {
			log.Info().Msgf("Cluster role %s found", role.Name)
			return true
		}
	}
	return false
}

func deleteClusterRole(m *e2e.Minikube, name string) {
	err := m.GetClient().RbacV1().ClusterRoles().Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to delete cluster role %s", name)
	} else {
		log.Info().Msgf("Cluster role %s deleted", name)
	}
}

// Helper function to measure request latency with optional HTTP method and headers
// If path is empty, "/" will be used
// If method is empty, "GET" will be used
func measureRequestLatency(m *e2e.Minikube, service metav1.Object, hostname string, path ...string) (time.Duration, error) {
	requestPath := "/"
	if len(path) > 0 && path[0] != "" {
		requestPath = path[0]
	}

	return measureRequestLatencyWithOptions(m, service, hostname, requestPath, "", nil)
}

// Helper function to measure request latency with custom HTTP method and headers
func measureRequestLatencyWithOptions(m *e2e.Minikube, service metav1.Object, hostname, path, method string, headers map[string]string) (time.Duration, error) {
	var (
		maxRetries = 8
		baseDelay  = 500 * time.Millisecond
	)

	if method == "" {
		method = "GET" // Default to GET if not specified
	}

	if path == "" {
		path = "/" // Default to root path if not specified
	}

	var diff time.Duration
	for attempt := 1; attempt <= maxRetries; attempt++ {
		client, err2 := m.NewRestClientForService(service)
		if err2 != nil {
			log.Error().Err(err2).Msg("Failed to create REST client")
			if attempt == maxRetries {
				return 0, err2
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}
		defer client.Close()

		// Set host header
		client.SetHeader("Host", hostname)

		// Set custom headers if provided
		for k, v := range headers {
			client.SetHeader(k, v)
		}

		// Prepare request
		request := client.R()

		// Make request with specified method
		startTime := time.Now()
		var resp *resty.Response
		var err error

		switch strings.ToUpper(method) {
		case "GET":
			resp, err = request.Get(path)
		case "POST":
			resp, err = request.Post(path)
		case "PUT":
			resp, err = request.Put(path)
		case "DELETE":
			resp, err = request.Delete(path)
		case "PATCH":
			resp, err = request.Patch(path)
		default:
			resp, err = request.Execute(method, path)
		}

		endTime := time.Now()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to make %s request", method)
			if attempt == maxRetries {
				return 0, err
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}
		diff = endTime.Sub(startTime)
		log.Info().Msgf("%s request to %s took %v (status: %d)", method, path, diff, resp.StatusCode())
		return diff, nil
	}
	return 0, fmt.Errorf("failed to measure request latency after %d attempts", maxRetries)
}

func checkStatusCode(t *testing.T, m *e2e.Minikube, service metav1.Object, path string, expectedStatusCode int, headers ...map[string]string) error {
	var (
		maxRetries = 8
		baseDelay  = 500 * time.Millisecond
	)

	// Check if headers are provided
	var requestHeaders map[string]string
	if len(headers) > 0 {
		requestHeaders = headers[0]
	}

	hasHeaders := len(requestHeaders) > 0

	for attempt := 1; attempt <= maxRetries; attempt++ {
		client, err := m.NewRestClientForService(service)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create REST client")
			if attempt == maxRetries {
				return err
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}
		defer client.Close()

		// Create request and add headers if provided
		request := client.R()
		if hasHeaders {
			for name, value := range requestHeaders {
				request.SetHeader(name, value)
			}
		}

		response, err := request.Get(path)
		if err != nil {
			log.Error().Err(err).Msg("Failed to make request")
			if attempt == maxRetries {
				return err
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}

		if response.StatusCode() != expectedStatusCode {
			if hasHeaders {
				log.Error().Msgf("Expected status code %d, got %d with headers %v",
					expectedStatusCode, response.StatusCode(), requestHeaders)
			} else {
				log.Error().Msgf("Expected status code %d, got %d",
					expectedStatusCode, response.StatusCode())
			}

			if attempt == maxRetries {
				return fmt.Errorf("expected status code %d, got %d", expectedStatusCode, response.StatusCode())
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}

		// Customize assertion message based on whether headers are used
		if hasHeaders {
			assert.Equal(t, expectedStatusCode, response.StatusCode(),
				"Expected status code %d, got %d with headers %v", expectedStatusCode, response.StatusCode(), requestHeaders)
			log.Info().Msgf("Request with headers %v returned status code %d as expected", requestHeaders, response.StatusCode())
		} else {
			assert.Equal(t, expectedStatusCode, response.StatusCode(),
				"Expected status code %d, got %d", expectedStatusCode, response.StatusCode())
			log.Info().Msgf("Request returned status code %d", response.StatusCode())
		}

		return nil
	}

	return fmt.Errorf("failed to get expected status code after %d attempts", maxRetries)
}

func int32Ptr(i int32) *int32 {
	return &i
}

func pathTypePtr(pathType networkingv1.PathType) *networkingv1.PathType {
	return &pathType
}

func testNginxIngressDiscovery(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	if isUsingRoleBinding() {
		log.Info().Msg("Skipping testNginxIngressDiscovery because it is using role binding, and is therefore not supported")
		return
	}
	log.Info().Msg("Starting testNginxIngressDiscovery")
	const nginxControllerNamespace = "nginx-controller"
	ctx, cancel := context.WithTimeout(context.Background(), 1800*time.Second)
	defer cancel()

	// Initialize NGINX Ingress Controller and test resources
	nginxService, testAppName, _, appDeployment, appService, appIngress := initNginxIngress(t, m, e, ctx, nginxControllerNamespace, "", "")
	defer func() { _ = m.DeleteDeployment(appDeployment) }()
	defer func() { _ = m.DeleteService(appService) }()
	defer func() { _ = m.DeleteIngress(appIngress) }()
	defer func() {
		cleanupNginxIngress(m, nginxControllerNamespace)
	}()

	// Test that we can find the NGINX Ingress target
	nginx, err := e2e.PollForTarget(ctx, e, extingress.NginxIngressTargetType, func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "k8s.ingress", testAppName)
	})
	require.NoError(t, err)
	assert.Equal(t, nginx.TargetType, extingress.NginxIngressTargetType)
	assert.Equal(t, nginx.Attributes["k8s.ingress"][0], testAppName)
	assert.Contains(t, nginx.Attributes["k8s.ingress.controller"][0], "ingress-nginx")
	assert.Equal(t, nginx.Attributes["k8s.ingress.class"][0], "nginx-steadybit")

	// Verify the application is accessible through NGINX Ingress
	err = checkStatusCode(t, m, nginxService, "/", 200)
	require.NoError(t, err)
	log.Info().Msg("Application is accessible through NGINX Ingress")
}

func testNginxBlockTraffic(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	if isUsingRoleBinding() {
		log.Info().Msg("Skipping testNginxBlockTraffic because it is using role binding, and is therefore not supported")
		return
	}
	log.Info().Msg("Starting testNginxBlockTraffic")
	const nginxControllerNamespace = "nginx-controller"
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Initialize NGINX Ingress Controller and test resources
	nginxService, _, ingressTarget, appDeployment, appService, appIngress := initNginxIngress(t, m, e, ctx, nginxControllerNamespace, "", "")
	defer func() { _ = m.DeleteDeployment(appDeployment) }()
	defer func() { _ = m.DeleteService(appService) }()
	defer func() { _ = m.DeleteIngress(appIngress) }()
	defer func() {
		cleanupNginxIngress(m, nginxControllerNamespace)
	}()

	// Verify the application is accessible before starting tests
	err := checkStatusCode(t, m, nginxService, "/", 200)
	require.NoError(t, err)
	log.Info().Msg("Application is accessible through NGINX Ingress before tests")

	// Define test cases
	tests := []struct {
		name                 string
		testPath             string
		responseStatusCode   int
		conditionPathPattern string
		conditionHttpMethod  string
		conditionHttpHeader  []interface{}
		requestHeaders       map[string]string
		isEnterpriseNginx    bool
		wantedBlock          bool
	}{
		{
			name:                 "should block traffic for the specified path",
			testPath:             "/",
			conditionPathPattern: "/",
			responseStatusCode:   503,
			wantedBlock:          true,
		},
		{
			name:                 "should block traffic for a specific endpoint",
			testPath:             "/",
			conditionPathPattern: "/api",
			responseStatusCode:   200,
			wantedBlock:          false, // We're requesting /, so /api block shouldn't affect us
		},
		{
			name:                "should block traffic for a http method",
			testPath:            "/",
			conditionHttpMethod: "GET",
			responseStatusCode:  503,
			wantedBlock:         true,
		},
		{
			name:                "should not block traffic for a http method",
			testPath:            "/",
			conditionHttpMethod: "DELETE",
			responseStatusCode:  200,
			wantedBlock:         false, // We're requesting DELETE, so / block shouldn't affect us
		},
		{
			name:                 "should block traffic for a http method and path",
			testPath:             "/",
			conditionHttpMethod:  "GET",
			conditionPathPattern: "/",
			responseStatusCode:   501,
			wantedBlock:          true,
		},
		{
			name:                 "should not block traffic for a http method and path",
			testPath:             "/",
			conditionHttpMethod:  "DELETE",
			conditionPathPattern: "/",
			responseStatusCode:   200,
			wantedBlock:          false, // We're requesting DELETE, so / block shouldn't affect us
		},
		{
			name:     "should block traffic for a specific header",
			testPath: "/",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
			},
			requestHeaders:     map[string]string{"User-Agent": "Mozilla/5.0"},
			responseStatusCode: 501,
			wantedBlock:        true,
		},
		{
			name:                 "should block traffic with combined header and path conditions",
			testPath:             "/",
			conditionPathPattern: "/",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "Content-Type", "value": "application/json"},
			},
			requestHeaders:     map[string]string{"Content-Type": "application/json"},
			responseStatusCode: 451,
			wantedBlock:        true,
		},
		{
			name:     "should not block traffic when header doesn't match",
			testPath: "/",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "X-Test-Header", "value": "specific-value"},
			},
			requestHeaders:     map[string]string{"X-Test-Header": "other-value"},
			responseStatusCode: 200,
			wantedBlock:        false, // We're not sending X-Test-Header
		},
		{
			name:                 "should block traffic with combined conditions",
			testPath:             "/",
			conditionPathPattern: "/",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "Content-Type", "value": "application/json"},
			},
			requestHeaders:     map[string]string{"Content-Type": "application/json"},
			responseStatusCode: 451,
			wantedBlock:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply block traffic action
			config := struct {
				Duration             int           `json:"duration"`
				ResponseStatusCode   int           `json:"responseStatusCode"`
				ConditionPathPattern string        `json:"conditionPathPattern"`
				ConditionHttpMethod  string        `json:"conditionHttpMethod"`
				ConditionHttpHeader  []interface{} `json:"conditionHttpHeader"`
				IsEnterpriseNginx    bool          `json:"isEnterpriseNginx"`
			}{
				Duration:             30000,
				ResponseStatusCode:   tt.responseStatusCode,
				ConditionPathPattern: tt.conditionPathPattern,
				ConditionHttpMethod:  tt.conditionHttpMethod,
				ConditionHttpHeader:  tt.conditionHttpHeader,
				IsEnterpriseNginx:    tt.isEnterpriseNginx,
			}

			//log.Info().Msgf("Applying NGINX block traffic action for path %s", tt.conditionPathtern)
			action, err := e.RunAction(extingress.NginxBlockTrafficActionId, ingressTarget, config, nil)
			require.NoError(t, err)
			defer func() { _ = action.Cancel() }()

			time.Sleep(5 * time.Second) // Give NGINX time to reconfigure

			// Verify block
			expectedStatusCode := tt.responseStatusCode
			if !tt.wantedBlock {
				expectedStatusCode = 200
			}

			// Check status code with headers if needed
			if len(tt.requestHeaders) > 0 {
				err = checkStatusCode(t, m, nginxService, tt.testPath, expectedStatusCode, tt.requestHeaders)
			} else {
				err = checkStatusCode(t, m, nginxService, tt.testPath, expectedStatusCode)
			}

			require.NoError(t, err)
			if tt.wantedBlock {
				log.Info().Msgf("Path %s is blocked with status %d as expected", tt.testPath, expectedStatusCode)
			} else {
				log.Info().Msgf("Path %s is not blocked, received status 200 as expected", tt.testPath)
			}

			// Cancel the action
			require.NoError(t, action.Cancel())

			time.Sleep(5 * time.Second) // Give NGINX time to reconfigure

			// Verify service is not blocked anymore
			err = checkStatusCode(t, m, nginxService, "/", 200)
			require.NoError(t, err, "Service should not be blocked anymore")
		})
	}
}

func initNginxIngress(t *testing.T, m *e2e.Minikube, e *e2e.Extension, ctx context.Context, nginxControllerNamespace string, imageName string, imageTag string) (*corev1.Service, string, *action_kit_api.Target, metav1.Object, metav1.Object, metav1.Object) {
	// Step 1: Deploy NGINX Ingress Controller
	log.Info().Msg("Deploying NGINX Ingress Controller")
	out, err := exec.Command("helm", "repo", "add", "ingress-nginx", "https://kubernetes.github.io/ingress-nginx").CombinedOutput()
	require.NoError(t, err, "Failed to add NGINX Helm repo: %s", out)
	out, err = exec.Command("helm", "repo", "update").CombinedOutput()
	require.NoError(t, err, "Failed to update Helm repos: %s", out)
	args := []string{
		"upgrade", "--install", "nginx-ingress", "ingress-nginx/ingress-nginx",
		"--create-namespace",
		"--namespace", nginxControllerNamespace,
		"--kube-context", m.Profile,
		"--set", "controller.service.type=NodePort",
		"--set", "controller.ingressClassResource.name=nginx-steadybit",
		"--set", "controller.ingressClass=nginx-steadybit",
		"--set", "controller.ingressClassResource.default=true",
		"--set", "controller.config.allow-snippet-annotations=true",
		"--set", "controller.config.annotations-risk-level=Critical",
		"--set", "controller.config.custom-http-snippet=\"load_module /etc/nginx/modules/ngx_steadybit_sleep_module.so;\"",
	}

	if imageName != "" {
		args = append(args, "--set", "controller.image.repository="+imageName)
		// Clear the digest to prevent SHA256 conflicts
		args = append(args, "--set", "controller.image.digest=")
	}
	if imageTag != "" {
		args = append(args, "--set", "controller.image.tag="+imageTag)
	}

	out, err = exec.Command("helm", args...).CombinedOutput()
	require.NoError(t, err, "Failed to deploy NGINX Ingress Controller: %s", out)

	// Wait for NGINX ingress controller to be ready
	log.Info().Msg("Waiting for NGINX Ingress Controller to be ready")
	err = waitForNginxIngressController(m, ctx, nginxControllerNamespace)
	require.NoError(t, err)

	// Step 2: Create a test deployment with service
	log.Info().Msg("Creating test deployment")
	testAppName := "nginx-ingress-test"

	// Create deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName,
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": testAppName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": testAppName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:stable-alpine",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	appDeployment, _, err := m.CreateDeployment(deployment)
	require.NoError(t, err)

	service := acorev1.Service(testAppName, "default").
		WithLabels(map[string]string{
			"app": testAppName,
		}).
		WithSpec(acorev1.ServiceSpec().
			WithSelector(map[string]string{
				"app": testAppName,
			}).
			WithPorts(acorev1.ServicePort().
				WithPort(80).
				WithTargetPort(intstr.FromInt32(80)),
			),
		)

	appService, err := m.CreateService(service)
	require.NoError(t, err)

	// Step 3: Create ingress resource
	log.Info().Msg("Creating ingress resource")
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName,
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx-steadybit",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: extutil.Ptr("nginx-steadybit"),
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pathTypePtr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: testAppName,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	appIngress, err := m.CreateIngress(ingress)
	require.NoError(t, err)

	// Get NGINX Ingress target
	log.Info().Msg("Finding NGINX Ingress target")
	var ingressTarget *action_kit_api.Target
	discoveryTarget, err := e2e.PollForTarget(ctx, e, extingress.NginxIngressTargetType, func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "k8s.ingress", testAppName)
	})
	require.NoError(t, err, "Failed to find NGINX ingress target")
	ingressTarget = &action_kit_api.Target{
		Attributes: discoveryTarget.Attributes,
	}

	// Find NGINX Service
	log.Info().Msg("Finding NGINX Service")
	nginxServiceName, err := findServiceNameInNamespace(m, nginxControllerNamespace)
	require.NoError(t, err, "Failed to find NGINX service")

	nginxService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxServiceName,
			Namespace: nginxControllerNamespace,
		},
	}

	return nginxService, testAppName, ingressTarget, appDeployment, appService, appIngress
}

func waitForNginxIngressController(m *e2e.Minikube, ctx context.Context, namespace string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			pods, err := m.GetClient().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Error().Err(err).Msgf("Failed to check NGINX controller status in namespace %s", namespace)
				continue
			}

			allReady := false
			for _, pod := range pods.Items {
				if strings.Contains(pod.Name, "controller") {
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if containerStatus.Ready && pod.Status.Phase == corev1.PodRunning {
							allReady = true
							break
						}
					}
					if allReady {
						break
					}
				}
			}

			if allReady {
				log.Info().Msgf("NGINX controller is ready in namespace %s", namespace)
				return nil
			}
			log.Info().Msgf("Waiting for NGINX controller in namespace %s to be ready...", namespace)
		}
	}
}

func cleanupNginxIngress(m *e2e.Minikube, nginxControllerNamespace string) {
	_ = exec.Command("helm", "uninstall", "nginx-ingress", "--namespace", nginxControllerNamespace, "--kube-context", m.Profile).Run()
	_ = exec.Command("kubectl", "--context", m.Profile, "delete", "namespace", nginxControllerNamespace, "--ignore-not-found").Run()
}

func testNginxDelayTraffic(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	if isUsingRoleBinding() {
		log.Info().Msg("Skipping testNginxDelayTraffic because it is using role binding, and is therefore not supported")
		return
	}
	log.Info().Msg("Starting testNginxDelayTraffic")
	const nginxNamespace = "nginx-ingress-steadybit"
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Initialize Ingress NGINX  Controller and test resources
	nginxService, testAppName, ingressTarget, appDeployment, appService, appIngress := initNginxIngress(t, m, e, ctx, nginxNamespace, "ghcr.io/steadybit/ingress-nginx-controller-with-steadybit-module", "main-community-v1.13.0")
	defer func() { _ = m.DeleteDeployment(appDeployment) }()
	defer func() { _ = m.DeleteService(appService) }()
	defer func() { _ = m.DeleteIngress(appIngress) }()
	defer func() {
		cleanupNginxIngress(m, nginxNamespace)
	}()

	// Measure baseline latency
	baselineLatency, err := measureRequestLatency(m, nginxService, testAppName+".local")
	require.NoError(t, err)
	log.Info().Msgf("Baseline latency: %v", baselineLatency)

	// Define delay parameters
	delayMs := 500
	tests := []struct {
		name                 string
		responseDelay        int
		conditionPathPattern string
		conditionHttpMethod  string
		conditionHttpHeader  []interface{}
		requestPath          string
		requestHeaders       map[string]string
		requestMethod        string
		wantedDelay          bool
	}{
		{
			name:                 "should delay traffic for the specified path",
			requestPath:          "/",
			conditionPathPattern: "/",
			responseDelay:        delayMs,
			wantedDelay:          true,
		},
		{
			name:                 "should not delay traffic for mismatched path",
			requestPath:          "/",
			conditionPathPattern: "/api",
			responseDelay:        delayMs,
			wantedDelay:          false,
		},
		{
			name:                "should delay traffic for specified HTTP method",
			requestPath:         "/",
			responseDelay:       delayMs,
			conditionHttpMethod: "GET",
			wantedDelay:         true,
		},
		{
			name:                "should not delay traffic for mismatched HTTP method",
			requestPath:         "/",
			conditionHttpMethod: "POST",
			wantedDelay:         false,
		},
		{
			name:        "should delay traffic for specified HTTP header",
			requestPath: "/",
			requestHeaders: map[string]string{
				"User-Agent": "Mozilla/5.0",
			},
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
			},
			responseDelay: delayMs,
			wantedDelay:   true,
		},
		{
			name:        "should not delay traffic for mismatched HTTP header",
			requestPath: "/",
			requestHeaders: map[string]string{
				"User-Agent": "Chrome/90.0",
			},
			responseDelay: delayMs,
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "User-Agent", "value": "Mozilla.*"},
			},
			wantedDelay: false,
		},
		{
			name:          "should delay traffic for combined conditions (all match)",
			requestPath:   "/",
			requestMethod: "GET",
			requestHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			responseDelay:        delayMs,
			conditionPathPattern: "/",
			conditionHttpMethod:  "GET",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "Content-Type", "value": "application/json"},
			},
			wantedDelay: true,
		},
		{
			name:          "should not delay traffic for combined conditions (one mismatch)",
			requestPath:   "/",
			requestMethod: "GET", // Mismatch - config requires POST
			requestHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			responseDelay:        delayMs,
			conditionPathPattern: ".*",
			conditionHttpMethod:  "POST",
			conditionHttpHeader: []interface{}{
				map[string]interface{}{"key": "Content-Type", "value": "application/json"},
			},
			wantedDelay: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply delay traffic action
			config := struct {
				Duration             int           `json:"duration"`
				ResponseDelay        int           `json:"responseDelay"`
				ConditionPathPattern string        `json:"conditionPathPattern,omitempty"`
				ConditionHttpMethod  string        `json:"conditionHttpMethod,omitempty"`
				ConditionHttpHeader  []interface{} `json:"conditionHttpHeader,omitempty"`
			}{
				Duration:             30000,
				ResponseDelay:        tt.responseDelay,
				ConditionPathPattern: tt.conditionPathPattern,
				ConditionHttpMethod:  tt.conditionHttpMethod,
				ConditionHttpHeader:  tt.conditionHttpHeader,
			}

			log.Info().Msgf("Applying delay of %dms to path %s", tt.responseDelay, tt.conditionPathPattern)
			action, err := e.RunAction(extingress.NginxDelayTrafficActionId, ingressTarget, config, nil)
			require.NoError(t, err)
			defer func() { _ = action.Cancel() }()

			// Measure latency during delay
			time.Sleep(5 * time.Second) // Give NGINX time to reconfigure

			// Use the correct method and headers for the test
			var delayedLatency time.Duration
			if tt.requestMethod != "" || len(tt.requestHeaders) > 0 {
				delayedLatency, err = measureRequestLatencyWithOptions(
					m,
					nginxService,
					testAppName+".local",
					tt.requestPath,
					tt.requestMethod,
					tt.requestHeaders,
				)
			} else {
				delayedLatency, err = measureRequestLatency(m, nginxService, testAppName+".local")
			}
			require.NoError(t, err)
			log.Info().Msgf("Latency during delay test: %v", delayedLatency)

			// Verify delay
			if tt.wantedDelay {
				// Check that delay is applied (with some tolerance)
				minExpectedLatency := baselineLatency + time.Duration(delayMs-50)*time.Millisecond  // -50ms tolerance
				maxExpectedLatency := baselineLatency + time.Duration(delayMs+200)*time.Millisecond // +200ms tolerance for overhead
				assert.GreaterOrEqual(t, delayedLatency, minExpectedLatency, "Latency should increase by approximately the configured delay")
				assert.LessOrEqual(t, delayedLatency, maxExpectedLatency, "Latency should not be much higher than expected")
			} else {
				// Latency shouldn't change significantly
				maxExpectedLatency := baselineLatency + 100*time.Millisecond // Allow for some small variance
				assert.LessOrEqual(t, delayedLatency, maxExpectedLatency, "Latency should not increase significantly")
			}

			// Cancel the action
			require.NoError(t, action.Cancel())

			// Measure latency after cancellation
			time.Sleep(5 * time.Second) // Give NGINX time to reconfigure
			var afterLatency time.Duration
			if tt.requestMethod != "" || len(tt.requestHeaders) > 0 {
				afterLatency, err = measureRequestLatencyWithOptions(
					m,
					nginxService,
					testAppName+".local",
					tt.requestPath,
					tt.requestMethod,
					tt.requestHeaders,
				)
			} else {
				afterLatency, err = measureRequestLatency(m, nginxService, testAppName+".local")
			}
			require.NoError(t, err)
			log.Info().Msgf("Latency after cancellation: %v", afterLatency)

			// Verify latency returned to normal
			maxExpectedAfterLatency := baselineLatency + 100*time.Millisecond
			assert.LessOrEqual(t, afterLatency, maxExpectedAfterLatency, "Latency should return to normal after cancellation")
		})
	}
}

// testNginxMultipleControllers tests the nginx delay functionality and module validation.
// This test validates that the improved ValidateNginxSteadybitModule function works correctly.
func testNginxMultipleControllers(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	if isUsingRoleBinding() {
		log.Info().Msg("Skipping testNginxMultipleControllers because it is using role binding, and is therefore not supported")
		return
	}
	log.Info().Msg("Starting testNginxMultipleControllers")

	const nginxNamespace = "nginx-multi-test"
	ctx, cancel := context.WithTimeout(context.Background(), 60*6*time.Second)
	defer cancel()

	// Deploy nginx controller WITH steadybit module
	log.Info().Msg("Deploying NGINX Ingress Controller with steadybit module")
	out, err := exec.Command("helm", "repo", "add", "ingress-nginx", "https://kubernetes.github.io/ingress-nginx").CombinedOutput()
	require.NoError(t, err, "Failed to add NGINX Helm repo: %s", out)
	out, err = exec.Command("helm", "repo", "update").CombinedOutput()
	require.NoError(t, err, "Failed to update Helm repos: %s", out)

	args := []string{
		"upgrade", "--install", "nginx-steadybit", "ingress-nginx/ingress-nginx",
		"--create-namespace",
		"--namespace", nginxNamespace,
		"--kube-context", m.Profile,
		"--set", "controller.service.type=NodePort",
		"--set", "controller.ingressClassResource.name=nginx-steadybit",
		"--set", "controller.ingressClass=nginx-steadybit",
		"--set", "controller.config.allow-snippet-annotations=true",
		"--set", "controller.config.annotations-risk-level=Critical",
		"--set", "controller.config.custom-http-snippet=\"load_module /etc/nginx/modules/ngx_steadybit_sleep_module.so;\"",
		"--set", "controller.image.repository=ghcr.io/steadybit/ingress-nginx-controller-with-steadybit-module",
		"--set", "controller.image.tag=main-community-v1.13.0",
		"--set", "controller.image.digest=",
		"--wait",
		"--timeout=180s",
	}
	log.Info().Msgf("Running helm command: helm %s", strings.Join(args, " "))
	out, err = exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		log.Error().Msgf("Helm output: %s", string(out))
	}
	require.NoError(t, err, "Failed to deploy NGINX Ingress Controller: %s", out)
	defer func() {
		_ = exec.Command("helm", "uninstall", "nginx-steadybit", "--namespace", nginxNamespace, "--kube-context", m.Profile).Run()
		_ = exec.Command("kubectl", "--context", m.Profile, "delete", "namespace", nginxNamespace, "--ignore-not-found").Run()
	}()

	// Helm --wait should have ensured the controller is ready, but let's double-check
	log.Info().Msg("Verifying NGINX controller is ready")
	err = waitForNginxIngressController(m, ctx, nginxNamespace)
	require.NoError(t, err)

	// Create test deployment and service
	log.Info().Msg("Creating test deployment")
	testAppName := "nginx-multi-controller-test"

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName,
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": testAppName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": testAppName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:stable-alpine",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	appDeployment, _, err := m.CreateDeployment(deployment)
	require.NoError(t, err)
	defer func() { _ = m.DeleteDeployment(appDeployment) }()

	service := acorev1.Service(testAppName, "default").
		WithLabels(map[string]string{
			"app": testAppName,
		}).
		WithSpec(acorev1.ServiceSpec().
			WithSelector(map[string]string{
				"app": testAppName,
			}).
			WithPorts(acorev1.ServicePort().
				WithPort(80).
				WithTargetPort(intstr.FromInt32(80)),
			),
		)

	appService, err := m.CreateService(service)
	require.NoError(t, err)
	defer func() { _ = m.DeleteService(appService) }()

	// Create ingress with steadybit class
	log.Info().Msg("Creating ingress resource targeting nginx-steadybit class")
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName,
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx-steadybit",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: extutil.Ptr("nginx-steadybit"),
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pathTypePtr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: testAppName,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	appIngress, err := m.CreateIngress(ingress)
	require.NoError(t, err)
	defer func() { _ = m.DeleteIngress(appIngress) }()

	// Wait for ingress target to be discovered
	log.Info().Msg("Finding NGINX Ingress target for steadybit class")
	var ingressTarget *action_kit_api.Target
	discoveryTarget, err := e2e.PollForTarget(ctx, e, extingress.NginxIngressTargetType, func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "k8s.ingress", testAppName) &&
			e2e.HasAttribute(target, "k8s.ingress.class", "nginx-steadybit")
	})
	require.NoError(t, err, "Failed to find NGINX ingress target with nginx-steadybit class")
	ingressTarget = &action_kit_api.Target{
		Attributes: discoveryTarget.Attributes,
	}

	// Find NGINX Service for delay measurement
	log.Info().Msg("Finding NGINX Service for delay measurement")
	nginxServiceName, err := findServiceNameInNamespace(m, nginxNamespace)
	require.NoError(t, err, "Failed to find NGINX service")
	nginxService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxServiceName,
			Namespace: nginxNamespace,
		},
	}

	// Measure baseline latency
	log.Info().Msg("Measuring baseline latency")
	baselineLatency, err := measureRequestLatency(m, nginxService, testAppName+".local")
	require.NoError(t, err)
	log.Info().Msgf("Baseline latency: %v", baselineLatency)

	// Test delay traffic with module validation
	log.Info().Msg("Testing nginx delay traffic with module validation on multiple replicas")
	delayMs := 1000 // 1 second delay
	config := struct {
		Duration             int    `json:"duration"`
		ResponseDelay        int    `json:"responseDelay"`
		ConditionPathPattern string `json:"conditionPathPattern"`
	}{
		Duration:             15000, // 15 seconds
		ResponseDelay:        delayMs,
		ConditionPathPattern: "/", // Match all paths - required condition
	}

	// This tests that validation works correctly
	action, err := e.RunAction(extingress.NginxDelayTrafficActionId, ingressTarget, config, nil)
	require.NoError(t, err, "Action should succeed when nginx controller has steadybit module")
	defer func() {
		if action != nil {
			_ = action.Cancel()
		}
	}()

	log.Info().Msg("Successfully validated nginx controller with steadybit module")

	// Measure latency during delay
	time.Sleep(5 * time.Second) // Give NGINX time to reconfigure
	delayedLatency, err := measureRequestLatency(m, nginxService, testAppName+".local")
	require.NoError(t, err)
	log.Info().Msgf("Latency during delay test: %v", delayedLatency)

	// Verify delay is applied (with tolerance)
	minExpectedLatency := baselineLatency + time.Duration(delayMs-50)*time.Millisecond  // -50ms tolerance
	maxExpectedLatency := baselineLatency + time.Duration(delayMs+200)*time.Millisecond // +200ms tolerance for overhead
	assert.GreaterOrEqual(t, delayedLatency, minExpectedLatency, "Latency should increase by approximately the configured delay")
	assert.LessOrEqual(t, delayedLatency, maxExpectedLatency, "Latency should not be much higher than expected")

	// Cancel the action
	require.NoError(t, action.Cancel())

	// Measure latency after cancellation
	time.Sleep(5 * time.Second) // Give NGINX time to reconfigure
	afterLatency, err := measureRequestLatency(m, nginxService, testAppName+".local")
	require.NoError(t, err)
	log.Info().Msgf("Latency after cancellation: %v", afterLatency)

	// Verify latency returned to normal
	maxExpectedAfterLatency := baselineLatency + 100*time.Millisecond
	assert.LessOrEqual(t, afterLatency, maxExpectedAfterLatency, "Latency should return to normal after cancellation")
}
