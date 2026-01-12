// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extingress

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

type RequestMatcher struct {
	PathPattern string
	HttpMethod  string
	HttpHeader  map[string]string
}

func parseRequestMatcher(config map[string]interface{}) (RequestMatcher, error) {
	var matcher RequestMatcher
	var err error

	matcher.PathPattern = extutil.ToString(config["conditionPathPattern"])
	matcher.HttpMethod = extutil.ToString(config["conditionHttpMethod"])

	if config["conditionHttpHeader"] != nil {
		matcher.HttpHeader, err = extutil.ToKeyValue(config, "conditionHttpHeader")
		if err != nil {
			return matcher, fmt.Errorf("failed to parse HTTP header condition: %w", err)
		}
	}

	// Validate that at least one condition is specified
	if matcher.PathPattern == "" && matcher.HttpMethod == "" && len(matcher.HttpHeader) == 0 {
		return matcher, fmt.Errorf("at least one condition (path, method, or header) is required")
	}

	return matcher, nil
}

func getCommonActionDescription(targetType, id, label, description, icon string) action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          id,
		Label:       label,
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: description,
		Technology:  extutil.Ptr("Kubernetes"),
		Icon:        extutil.Ptr(icon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: targetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "ingress",
					Description: extutil.Ptr("Find ingress by cluster, namespace and ingress"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.ingress=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Description:  extutil.Ptr("The duration of the action. The ingress will be affected for the specified duration."),
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
		},
	}
}
