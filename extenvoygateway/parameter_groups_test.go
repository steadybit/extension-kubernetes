// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	"testing"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/stretchr/testify/assert"
)

func TestParameterGroups(t *testing.T) {
	abort := getAbortDescription()
	assert.Equal(t, map[string][]string{
		groupResponseDetails: {"body", "contentType"},
		groupTrafficFilter:   {"sectionName"},
	}, groupMembers(abort))
	assert.Equal(t, []string{groupTrafficFilter}, definedGroups(abort))
	assertNoAdvancedWithoutGroup(t, abort)
	assertTrafficFilterLast(t, abort)

	delay := getDelayDescription()
	assert.Equal(t, map[string][]string{groupTrafficFilter: {"sectionName"}}, groupMembers(delay))
	assert.Equal(t, []string{groupTrafficFilter}, definedGroups(delay))
	assertNoAdvancedWithoutGroup(t, delay)
	assertTrafficFilterLast(t, delay)
}

func groupMembers(desc action_kit_api.ActionDescription) map[string][]string {
	members := map[string][]string{}
	for _, p := range desc.Parameters {
		if p.Group != nil {
			members[*p.Group] = append(members[*p.Group], p.Name)
		}
	}
	return members
}

func definedGroups(desc action_kit_api.ActionDescription) []string {
	var names []string
	if desc.ParameterGroups != nil {
		for _, g := range *desc.ParameterGroups {
			names = append(names, g.Name)
		}
	}
	return names
}

// assertNoAdvancedWithoutGroup checks that no advanced parameter relies on the platform fallback for "Additional Settings".
func assertNoAdvancedWithoutGroup(t *testing.T, desc action_kit_api.ActionDescription) {
	for _, p := range desc.Parameters {
		if p.Advanced != nil && *p.Advanced {
			assert.NotNil(t, p.Group, "parameter %s", p.Name)
		}
	}
}

// assertTrafficFilterLast checks that the route rule filter is the last parameter: these actions set no order, so
// platforms show the parameters in list order and a group where its first parameter is.
func assertTrafficFilterLast(t *testing.T, desc action_kit_api.ActionDescription) {
	last := desc.Parameters[len(desc.Parameters)-1]
	assert.Equal(t, "sectionName", last.Name)
	assert.True(t, last.Advanced != nil && *last.Advanced, "platforms without groups keep it in Additional Settings")
	assert.Equal(t, groupTrafficFilter, *last.Group)
	for _, p := range desc.Parameters {
		assert.Nil(t, p.Order, "parameter %s: the list order decides", p.Name)
	}
}
