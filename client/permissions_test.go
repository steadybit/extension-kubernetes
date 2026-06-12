// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsArgoRolloutScalePermitted(t *testing.T) {
	permitted := &PermissionCheckResult{
		Permissions: map[string]PermissionCheckOutcome{
			"argoproj.io/rollouts/scale/get":    OK,
			"argoproj.io/rollouts/scale/update": OK,
			"argoproj.io/rollouts/scale/patch":  OK,
		},
	}
	assert.True(t, permitted.IsArgoRolloutScalePermitted())

	missing := &PermissionCheckResult{
		Permissions: map[string]PermissionCheckOutcome{
			"argoproj.io/rollouts/scale/get": OK,
		},
	}
	assert.False(t, missing.IsArgoRolloutScalePermitted())
}
