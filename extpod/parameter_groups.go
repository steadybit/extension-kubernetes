// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extpod

import "github.com/steadybit/action-kit/go/action_kit_api/v2"

const groupCrashBehavior = "Crash Behavior"

var crashBehaviorGroup = action_kit_api.ParameterGroup{
	Name:        groupCrashBehavior,
	Description: "The signal goes to the main process (PID 1) each time the container runs again. SIGKILL and SIGSTOP are not allowed.",
}
