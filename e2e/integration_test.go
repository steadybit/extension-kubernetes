// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kubernetes/extcluster"
	"github.com/steadybit/extension-kubernetes/extcontainer"
	"github.com/steadybit/extension-kubernetes/extdeployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestWithMinikube(t *testing.T) {
	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-kubernetes",
		Port: 8088,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", "kubernetes.clusterName=e2e-cluster",
				//"--set", "logging.level=debug",
			}
		},
	}

	e2e.WithDefaultMinikube(t, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "discovery",
			Test: testDiscovery,
		},
	})
}

func testDiscovery(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testDiscovery")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nginxDeployment := e2e.NginxDeployment{Minikube: m}
	err := nginxDeployment.Deploy("nginx")
	require.NoError(t, err, "failed to create deployment")
	defer func() { _ = nginxDeployment.Delete() }()

	target, err := e2e.PollForTarget(ctx, e, extdeployment.DeploymentTargetType, func(target discovery_kit_api.Target) bool {
		log.Debug().Msgf("deployment: %v", target.Attributes["k8s.deployment"])
		return e2e.HasAttribute(target, "k8s.deployment", "nginx")
	})

	require.NoError(t, err)
	assert.Equal(t, target.TargetType, extdeployment.DeploymentTargetType)
	assert.Equal(t, target.Attributes["k8s.namespace"][0], "default")
	assert.Equal(t, target.Attributes["k8s.deployment"][0], "nginx")
	assert.Equal(t, target.Attributes["k8s.deployment.label.app"][0], "nginx")
	assert.Equal(t, target.Attributes["k8s.cluster-name"][0], "e2e-cluster")
	assert.Equal(t, target.Attributes["k8s.pod.name"][0], nginxDeployment.Pods[0].Name)
	assert.Equal(t, target.Attributes["k8s.distribution"][0], "kubernetes")

	target, err = e2e.PollForTarget(ctx, e, extcontainer.KubernetesContainerTargetType, func(target discovery_kit_api.Target) bool {
		log.Debug().Msgf("target: %v", target.Attributes["k8s.container.name"])
		return e2e.HasAttribute(target, "k8s.container.name", "nginx")
	})

	require.NoError(t, err)
	assert.Equal(t, target.TargetType, extcontainer.KubernetesContainerTargetType)
	assert.Equal(t, target.Attributes["k8s.container.name"][0], "nginx")
	assert.Equal(t, target.Attributes["k8s.container.ready"][0], "true")
	assert.Equal(t, target.Attributes["k8s.container.image"][0], "nginx:stable-alpine")
	assert.Equal(t, target.Attributes["k8s.pod.label.app"][0], "nginx")
	assert.Equal(t, target.Attributes["k8s.namespace"][0], "default")
	assert.Equal(t, target.Attributes["k8s.node.name"][0], "e2e-docker")
	assert.Equal(t, target.Attributes["k8s.pod.name"][0], nginxDeployment.Pods[0].Name)
	assert.Equal(t, target.Attributes["k8s.distribution"][0], "kubernetes")

	target, err = e2e.PollForTarget(ctx, e, extcluster.ClusterTargetType, func(target discovery_kit_api.Target) bool {
		log.Debug().Msgf("target: %v", target.Attributes["k8s.cluster-name"])
		return e2e.HasAttribute(target, "k8s.cluster-name", "e2e-cluster")
	})

	require.NoError(t, err)
	assert.Equal(t, target.TargetType, extcluster.ClusterTargetType)
}
