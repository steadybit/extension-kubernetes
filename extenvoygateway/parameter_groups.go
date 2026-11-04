// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import "github.com/steadybit/action-kit/go/action_kit_api/v2"

const (
	groupResponseDetails = "Response Details"
	groupTrafficFilter   = "Traffic Filter"
)

// The query can select several routes; the route rule name narrows each of them to one named rule.
var trafficFilterGroup = action_kit_api.ParameterGroup{
	Name:        groupTrafficFilter,
	Description: "Leave unchanged to affect all rules of the selected routes.",
}
