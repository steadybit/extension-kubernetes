// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"

	"fmt"
	"os"
	"os/exec"
	"strconv"
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
}

func TestWithMinikube(t *testing.T) {
	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-kubernetes",
		Port: 8088,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", "kubernetes.clusterName=e2e-cluster",
				"--set", "discovery.attributes.excludes.container={k8s.label.*}",
				"--set", "logging.level=debug",
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
				"--set", "logging.level=debug",
				"--set", "role.create=true",
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
		name        string
		path        string
		delay       int
		wantedDelay bool
	}{
		{
			name:        "should delay traffic for the specified path",
			path:        "/",
			delay:       delayMs,
			wantedDelay: true,
		},
		{
			name:        "should delay traffic for a specific endpoint",
			path:        "/api",
			delay:       delayMs,
			wantedDelay: false, // We're requesting /, so /api delay shouldn't affect us
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply delay traffic action
			config := struct {
				Duration int    `json:"duration"`
				Path     string `json:"path"`
				Delay    int    `json:"delay"`
			}{
				Duration: 30000,
				Path:     tt.path,
				Delay:    tt.delay,
			}

			log.Info().Msgf("Applying delay of %dms to path %s", tt.delay, tt.path)
			action, err := e.RunAction(extingress.HAProxyDelayTrafficActionId, ingressTarget, config, nil)
			require.NoError(t, err)
			defer func() { _ = action.Cancel() }()

			// Measure latency during delay
			time.Sleep(5 * time.Second) // Give HAProxy time to reconfigure
			delayedLatency, err := measureRequestLatency(m, haProxyService, testAppName+".local")
			require.NoError(t, err)
			log.Info().Msgf("Latency during delay: %v", delayedLatency)

			// Verify delay
			if tt.wantedDelay {
				// Check that delay is applied (with some tolerance)
				minExpectedLatency := baselineLatency + time.Duration(tt.delay-50)*time.Millisecond  // -50ms tolerance
				maxExpectedLatency := baselineLatency + time.Duration(tt.delay+200)*time.Millisecond // +200ms tolerance for overhead
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
			afterLatency, err := measureRequestLatency(m, haProxyService, testAppName+".local")
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
		name           string
		testPath       []string
		pathStatusCode []interface{}
		wantedBlock    bool
	}{
		{
			name:     "should block traffic for the specified path",
			testPath: []string{"/"},
			pathStatusCode: []interface{}{
				map[string]interface{}{"key": "/", "value": "503"},
			},
			wantedBlock: true,
		},
		{
			name:     "should block traffic for a specific endpoint",
			testPath: []string{"/"},
			pathStatusCode: []interface{}{
				map[string]interface{}{"key": "/api", "value": "503"},
			},
			wantedBlock: false, // We're requesting /, so /api block shouldn't affect us
		},
		{
			name:     "should block traffic for multiple endpoints",
			testPath: []string{"/block", "/api"},
			pathStatusCode: []interface{}{
				map[string]interface{}{"key": "/block", "value": "503"},
				map[string]interface{}{"key": "/api", "value": "401"},
			},
			wantedBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply delay traffic action
			config := struct {
				Duration       int           `json:"duration"`
				PathStatusCode []interface{} `json:"pathStatusCode"`
			}{
				Duration:       30000,
				PathStatusCode: tt.pathStatusCode,
			}

			log.Info().Msgf("Applying block to path %s", tt.pathStatusCode)
			action, err := e.RunAction(extingress.HAProxyBlockTrafficActionId, ingressTarget, config, nil)
			require.NoError(t, err)
			defer func() { _ = action.Cancel() }()

			time.Sleep(5 * time.Second) // Give HAProxy time to reconfigure

			// Verify block
			for _, path := range tt.testPath {
				var expectedStatusCode int
				for _, pathStatus := range tt.pathStatusCode {
					if pathStatus.(map[string]interface{})["key"] == path {
						expectedStatusCode, err = strconv.Atoi(pathStatus.(map[string]interface{})["value"].(string))
						require.NoError(t, err)
						break
					}
				}
				if expectedStatusCode == 0 {
					if tt.wantedBlock {
						log.Error().Msgf("Expected status code not found for path %s", path)
						t.Fail()
					} else {
						expectedStatusCode = 200
					}
				}
				if tt.wantedBlock {
					err = checkStatusCode(t, m, haProxyService, path, expectedStatusCode)
					require.NoError(t, err)
					log.Info().Msgf("Path %s is blocked", path)
				} else {
					err = checkStatusCode(t, m, haProxyService, "", 200)
					require.NoError(t, err)
				}
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

// initHAProxy initializes HAProxy controller and test resources
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

func measureRequestLatency(m *e2e.Minikube, service metav1.Object, hostname string) (time.Duration, error) {
	var (
		maxRetries = 8
		baseDelay  = 500 * time.Millisecond
	)

	var diff time.Duration
	for attempt := 1; attempt <= maxRetries; attempt++ {
		client, err := m.NewRestClientForService(service)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create REST client")
			if attempt == maxRetries {
				return 0, err
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}
		defer client.Close()
		client.SetHeader("Host", hostname)
		startTime := time.Now()
		_, err = client.R().Get("/")
		endTime := time.Now()
		if err != nil {
			log.Error().Err(err).Msg("Failed to make request")
			if attempt == maxRetries {
				return 0, err
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}
		diff = endTime.Sub(startTime)
		log.Info().Msgf("Request took %v", diff)
		return diff, nil
	}
	return 0, fmt.Errorf("failed to measure request latency after %d attempts", maxRetries)
}

func checkStatusCode(t *testing.T, m *e2e.Minikube, service metav1.Object, path string, expectedStatusCode int) error {
	var (
		maxRetries = 8
		baseDelay  = 500 * time.Millisecond
	)
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
		response, err := client.R().Get(path)
		if err != nil {
			log.Error().Err(err).Msg("Failed to make request")
			if attempt == maxRetries {
				return err
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}
		if response.StatusCode() != expectedStatusCode {
			log.Error().Msgf("Expected status code %d, got %d", expectedStatusCode, response.StatusCode())
			if attempt == maxRetries {
				return fmt.Errorf("expected status code %d, got %d", expectedStatusCode, response.StatusCode())
			}
			time.Sleep(baseDelay * (1 << (attempt - 1)))
			continue
		}
		assert.Equal(t, response.StatusCode(), expectedStatusCode, "Expected status code %d, got %d", response.StatusCode(), expectedStatusCode)
		log.Info().Msgf("Request returned status code %d", response.StatusCode())
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
