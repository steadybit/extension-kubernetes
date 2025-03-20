// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcluster

import (
	"context"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_getDiscoveredCluster(t *testing.T) {
	// Given
	extconfig.Config.ClusterName = "dev-cluster"

	d := &clusterDiscovery{}

	//Then
	targets, _ := d.DiscoverTargets(context.Background())
	require.Len(t, targets, 1)
	target := targets[0]
	assert.Equal(t, "dev-cluster", target.Id)
	assert.Equal(t, "dev-cluster", target.Label)
	assert.Equal(t, ClusterTargetType, target.TargetType)
	assert.Equal(t, map[string][]string{
		"k8s.cluster-name": {"dev-cluster"},
	}, target.Attributes)
}
