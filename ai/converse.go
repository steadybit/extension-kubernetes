package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// ConverseWrapper encapsulates Amazon Bedrock actions used in the examples.
type ConverseWrapper struct {
	BedrockRuntimeClient BedrockConverseClient
}

type BedrockConverseClient interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

// FindReliabilityIssues is the updated Go equivalent of your Java method.
func (w ConverseWrapper) FindReliabilityIssues(
	ctx context.Context,
	technology string,
	targetType string,
	resourceJSON string,
	reliabilityPriority *string,
) (string, error) {

	modelID := "eu.anthropic.claude-sonnet-4-5-20250929-v1:0"
	riTool := GetReliabilityIssuesTool()
	systemPrompt := "You are a reliability assessor. You MUST call the tool named 'reliabilityIssues'."

	// Build system message
	systemBlocks := []types.SystemContentBlock{
		&types.SystemContentBlockMemberText{
			Value: systemPrompt,
		},
	}

	// Build the JSON payload sent as the USER message (same structure as in Java)
	priorityField := "null"
	if reliabilityPriority != nil {
		priorityField = fmt.Sprintf(`"%s"`, *reliabilityPriority)
	}

	userPayload := fmt.Sprintf(
		`{"technology":"%s","target_types":"%s","resource_json":%s,"reliability_priority":%s}`,
		technology,
		targetType,
		resourceJSON,
		priorityField,
	)

	fmt.Println("Approx input tokens:", EstimateTokens(userPayload))

	userMessage := types.Message{
		Role: types.ConversationRoleUser,
		Content: []types.ContentBlock{
			&types.ContentBlockMemberText{
				Value: userPayload,
			},
		},
	}

	var inputSchemaObj map[string]any
	if err := json.Unmarshal([]byte(riTool.SchemaJSON), &inputSchemaObj); err != nil {
		return "", fmt.Errorf("invalid reliabilityIssues tool schema JSON: %w", err)
	}

	toolSpec := types.ToolSpecification{
		Name:        aws.String(riTool.Name),
		Description: aws.String(riTool.Description),
		InputSchema: &types.ToolInputSchemaMemberJson{
			Value: document.NewLazyDocument(inputSchemaObj),
		},
	}

	// Tool configuration: force use of `reliabilityIssues` tool with your JSON schema
	toolConfig := &types.ToolConfiguration{
		Tools: []types.Tool{
			&types.ToolMemberToolSpec{
				Value: toolSpec,
			},
		},
	}

	// Optional: reasoning config (thinking) like in your Java code
	thinkingDoc := document.NewLazyDocument(map[string]any{
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": 1500,
		},
	})

	inferenceCfg := &types.InferenceConfiguration{
		MaxTokens: aws.Int32(2000),
	}

	input := &bedrockruntime.ConverseInput{
		ModelId:                      aws.String(modelID),
		System:                       systemBlocks,
		Messages:                     []types.Message{userMessage},
		ToolConfig:                   toolConfig,
		InferenceConfig:              inferenceCfg,
		AdditionalModelRequestFields: thinkingDoc,
	}

	resp, err := w.BedrockRuntimeClient.Converse(context.Background(), input)
	if err != nil {
		// You can plug your own ProcessError here if you like
		return "", fmt.Errorf("bedrock Converse error: %w", err)
	}

	// Extract the message output
	msgOut, ok := resp.Output.(*types.ConverseOutputMemberMessage)
	if !ok || msgOut == nil {
		return "", fmt.Errorf("unexpected Converse output type")
	}

	response, _ := resp.Output.(*types.ConverseOutputMemberMessage)

	var toolUseOutputJson []byte
	if len(response.Value.Content) > 0 {
		contentBlock := response.Value.Content[1]
		toolUseOutput, _ := contentBlock.(*types.ContentBlockMemberToolUse)
		if toolUseOutput.Value.Input != nil {
			toolUseOutputJson, err = toolUseOutput.Value.Input.MarshalSmithyDocument()
			if err != nil {
				return "", fmt.Errorf("unable to marshal tool use output, %v", err)
			}
		} else {
			return "", fmt.Errorf("AI Response is not containing content to parse")
		}
	} else {
		return "", fmt.Errorf("AI Response is not containing content to parse")
	}

	// Pretty-print JSON if possible (same behavior as Java)
	var prettyObj any
	var prettyPrintOut []byte
	if err := json.Unmarshal(toolUseOutputJson, &prettyObj); err == nil {
		prettyPrintOut, _ = json.MarshalIndent(prettyObj, "", "  ")
		fmt.Println("Pretty JSON response:\n" + string(prettyPrintOut))
	}
	fmt.Println("Approx output tokens:", EstimateTokens(string(prettyPrintOut)))

	return string(prettyPrintOut), nil
}
