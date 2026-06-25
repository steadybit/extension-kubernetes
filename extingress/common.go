// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extingress

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

type RequestMatcher struct {
	PathPattern string
	HttpMethod  string
	HttpHeader  map[string]string
}

func parseRequestMatcher(config map[string]any) (RequestMatcher, error) {
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

	// Matcher values are interpolated into nginx/haproxy config snippets. Reject control
	// characters (in particular newlines) so a value cannot break out of its directive and
	// inject additional configuration into the ingress controller.
	if containsControlChar(matcher.PathPattern) {
		return matcher, fmt.Errorf("path pattern must not contain control characters")
	}
	if containsControlChar(matcher.HttpMethod) {
		return matcher, fmt.Errorf("HTTP method must not contain control characters")
	}
	for name, value := range matcher.HttpHeader {
		if containsControlChar(name) || containsControlChar(value) {
			return matcher, fmt.Errorf("HTTP header condition must not contain control characters")
		}
	}

	return matcher, nil
}

func containsControlChar(value string) bool {
	return strings.IndexFunc(value, unicode.IsControl) >= 0
}

func getCommonActionDescription(targetType, id, label, description, icon string) action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          id,
		Label:       label,
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: description,
		Technology:  new("Kubernetes"),
		Icon:        new(icon),
		TargetSelection: new(action_kit_api.TargetSelection{
			TargetType: targetType,
			SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "ingress",
					Description: new("Find ingress by cluster, namespace and ingress"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.ingress=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Description:  new("The duration of the action. The ingress will be affected for the specified duration."),
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: new("30s"),
				Required:     new(true),
			},
		},
	}
}
