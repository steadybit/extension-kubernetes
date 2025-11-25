package ai

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// --- Mock Client ---

type mockBedrockClient struct {
	*bedrockruntime.Client
	ConverseFunc func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

func (m mockBedrockClient) Converse(
	ctx context.Context,
	params *bedrockruntime.ConverseInput,
	optFns ...func(*bedrockruntime.Options),
) (*bedrockruntime.ConverseOutput, error) {
	return m.ConverseFunc(ctx, params, optFns...)
}

// --- Test ---

func TestFindReliabilityIssues(t *testing.T) {
	// Mock tool output JSON
	toolOutput := map[string]any{
		"issues": []any{
			map[string]any{
				"id":          "ISSUE-1",
				"description": "Mock reliability issue",
			},
		},
	}

	mockClient := mockBedrockClient{
		ConverseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {

			// This is the "AI message" (required as first element)
			textBlock := &types.ContentBlockMemberText{
				Value: "Tool call executed.",
			}

			// Tool-use ContentBlock
			toolUse := &types.ContentBlockMemberToolUse{
				Value: types.ToolUseBlock{
					Name:  aws.String("reliabilityIssues"),
					Input: document.NewLazyDocument(toolOutput),
				},
			}

			return &bedrockruntime.ConverseOutput{
				Output: &types.ConverseOutputMemberMessage{
					Value: types.Message{
						Role: types.ConversationRoleAssistant,
						Content: []types.ContentBlock{
							textBlock,
							toolUse,
						},
					},
				},
			}, nil
		},
	}

	w := ConverseWrapper{
		BedrockRuntimeClient: mockClient,
	}

	result, err := w.FindReliabilityIssues(
		context.Background(),
		"kubernetes",
		"deployment",
		`{"kind":"Deployment"}`,
		nil,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output contains expected JSON structure
	if !json.Valid([]byte(result)) {
		t.Fatalf("expected valid JSON output, got: %s", result)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	issues, ok := parsed["issues"]
	if !ok {
		t.Fatalf("expected 'issues' field in result")
	}

	list, ok := issues.([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("expected non-empty issues array")
	}
}
