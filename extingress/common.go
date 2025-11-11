// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"fmt"

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
