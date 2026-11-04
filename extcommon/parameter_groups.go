// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import "github.com/steadybit/action-kit/go/action_kit_api/v2"

const groupLogFilter = "Log Filter"

var logFilterGroup = action_kit_api.ParameterGroup{
	Name:        groupLogFilter,
	Description: "Leave unchanged to count lines matching the default error pattern in all containers.",
}
