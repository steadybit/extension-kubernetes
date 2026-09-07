// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfiguration(t *testing.T) {
	original := Config
	t.Cleanup(func() { Config = original })
	t.Setenv("STEADYBIT_EXTENSION_CLUSTER_NAME", "development")
	t.Setenv("STEADYBIT_EXTENSION_NAMESPACE", "shop")
	t.Setenv("STEADYBIT_EXTENSION_DISABLE_DISCOVERY_EXCLUDES", "true")

	ParseConfiguration()
	ValidateConfiguration()

	assert.Equal(t, "development", Config.ClusterName)
	assert.True(t, Config.DisableDiscoveryExcludes)
	assert.Equal(t, 50, Config.DiscoveryMaxPodCount, "defaults must still apply")
	assert.True(t, HasNamespaceFilter())

	Config.Namespace = ""
	assert.False(t, HasNamespaceFilter())
}
