/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package client

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoveAnnotationBlock(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		startMarker string
		endMarker   string
		expected    string
	}{
		{
			name: "basic removal",
			config: `prefix text
# BEGIN STEADYBIT - abc123
some config to remove
more config
# END STEADYBIT - abc123
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "prefix text\nsuffix text",
		},
		{
			name: "markers not found",
			config: `prefix text
# BEGIN STEADYBIT - xyz789
some other config
# END STEADYBIT - xyz789
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected: `prefix text
# BEGIN STEADYBIT - xyz789
some other config
# END STEADYBIT - xyz789
suffix text`,
		},
		{
			name: "only start marker",
			config: `prefix text
# BEGIN STEADYBIT - abc123
some config
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected: `prefix text
# BEGIN STEADYBIT - abc123
some config
suffix text`,
		},
		{
			name: "only end marker",
			config: `prefix text
some config
# END STEADYBIT - abc123
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected: `prefix text
some config
# END STEADYBIT - abc123
suffix text`,
		},
		{
			name: "with trailing newlines",
			config: `prefix text
# BEGIN STEADYBIT - abc123
some config
# END STEADYBIT - abc123


suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "prefix text\nsuffix text",
		},
		{
			name: "multiple blocks",
			config: `prefix text
# BEGIN STEADYBIT - abc123
first block
# END STEADYBIT - abc123
middle text
# BEGIN STEADYBIT - abc123
second block
# END STEADYBIT - abc123
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected: `prefix text
middle text
# BEGIN STEADYBIT - abc123
second block
# END STEADYBIT - abc123
suffix text`,
		},
		{
			name:        "empty config",
			config:      "",
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "",
		},
		{
			name: "block at start",
			config: `# BEGIN STEADYBIT - abc123
some config
# END STEADYBIT - abc123
suffix text`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "suffix text",
		},
		{
			name: "block at end",
			config: `prefix text
# BEGIN STEADYBIT - abc123
some config
# END STEADYBIT - abc123`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "prefix text\n",
		},
		{
			name: "only the block",
			config: `# BEGIN STEADYBIT - abc123
some config
# END STEADYBIT - abc123`,
			startMarker: "# BEGIN STEADYBIT - abc123",
			endMarker:   "# END STEADYBIT - abc123",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeAnnotationBlock(tt.config, tt.startMarker, tt.endMarker)
			assert.Equal(t, tt.expected, result)
		})
	}
}
