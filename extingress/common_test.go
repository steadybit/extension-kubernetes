// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRequestMatcher(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]any
		wantErr string
	}{
		{
			name:   "accepts a regex path pattern",
			config: map[string]any{"conditionPathPattern": "/api/.*"},
		},
		{
			name:    "requires at least one condition",
			config:  map[string]any{},
			wantErr: "at least one condition (path, method, or header) is required",
		},
		{
			name:    "rejects a newline in the path pattern (config snippet injection)",
			config:  map[string]any{"conditionPathPattern": "/api\nreturn 200 \"injected\";"},
			wantErr: "path pattern must not contain control characters",
		},
		{
			name:    "rejects a newline in the http method",
			config:  map[string]any{"conditionHttpMethod": "GET\nmore-directives"},
			wantErr: "HTTP method must not contain control characters",
		},
		{
			name: "rejects a newline in a header value",
			config: map[string]any{"conditionHttpHeader": []any{
				map[string]any{"key": "X-Test", "value": "ok\ninjected"},
			}},
			wantErr: "HTTP header condition must not contain control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := parseRequestMatcher(tt.config)
			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.NotEmpty(t, matcher.PathPattern+matcher.HttpMethod)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
