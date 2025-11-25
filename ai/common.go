package ai

import "strings"

type Prompt struct {
	System string
	User   string
}

type ToolDefinition struct {
	Name        string
	Description string
	SchemaJSON  string
}

func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	chars := len([]rune(text))
	words := len(splitWords(text))

	byChars := (chars + 3) / 4
	byWords := int(float64(words)*1.1 + 0.5)

	estimate := max(byChars, byWords)
	if chars <= 8 && words <= 2 {
		return 1
	}
	return estimate
}

func splitWords(s string) []string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return []string{}
	}
	return parts
}
