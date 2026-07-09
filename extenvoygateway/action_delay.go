// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
)

func NewDelayAction(k8s *client.Client) action_kit_sdk.Action[ActionState] {
	return &backendTrafficPolicyAction{
		k8s:              k8s,
		description:      getDelayDescription(),
		subtype:          "delay",
		buildFaultSpecFn: buildDelayFaultSpec,
	}
}

func getDelayDescription() action_kit_api.ActionDescription {
	desc := getCommonActionDescription(
		DelayActionId,
		"Envoy Delay Traffic",
		"Inject a fixed delay into a percentage of the traffic on an Envoy Gateway HTTP route using a BackendTrafficPolicy.",
	)
	desc.Parameters = append(desc.Parameters, action_kit_api.ActionParameter{
		Name:         "delay",
		Label:        "Delay",
		Description:  new("The fixed delay to inject into matching requests."),
		Type:         action_kit_api.ActionParameterTypeDuration,
		DefaultValue: new("500ms"),
		Required:     new(true),
	})
	return withSectionNameParameter(desc)
}

func buildDelayFaultSpec(config map[string]any) (map[string]any, error) {
	delayMs := extutil.ToInt64(config["delay"])
	if delayMs <= 0 {
		return nil, fmt.Errorf("delay must be greater than zero")
	}
	return map[string]any{
		"faultInjection": map[string]any{
			"delay": map[string]any{
				"fixedDelay": fmt.Sprintf("%dms", delayMs),
				"percentage": percentageFromConfig(config),
			},
		},
	}, nil
}
