// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extingress

import (
	"testing"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/stretchr/testify/assert"
)

func TestParameterGroups(t *testing.T) {
	for name, desc := range map[string]action_kit_api.ActionDescription{
		"nginxBlock": getNginxBlockTrafficDescription(),
		"nginxDelay": getNginxDelayTrafficDescription(),
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, map[string][]string{action_kit_api.ParameterGroupAdditionalSettings: {"isEnterpriseNginx"}}, groupMembers(desc))
			assertNoAdvancedWithoutGroup(t, desc)
			assertAdditionalSettingsLast(t, desc, "isEnterpriseNginx")
		})
	}
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

// assertAdditionalSettingsLast checks that "Additional Settings" is the last parameter: these actions set no order, so
// platforms show the parameters in list order and a group where its first parameter is.
func assertAdditionalSettingsLast(t *testing.T, desc action_kit_api.ActionDescription, name string) {
	last := desc.Parameters[len(desc.Parameters)-1]
	assert.Equal(t, name, last.Name)
	assert.Equal(t, action_kit_api.ParameterGroupAdditionalSettings, *last.Group)
	for _, p := range desc.Parameters {
		assert.Nil(t, p.Order, "parameter %s: the list order decides", p.Name)
	}
}
