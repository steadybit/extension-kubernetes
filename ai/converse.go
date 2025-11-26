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

func (w ConverseWrapper) RecommendTemplate(
	issues string,
	templatesAvailable string,
) (string, error) {

	modelID := "eu.anthropic.claude-sonnet-4-5-20250929-v1:0"

	// Build the tool schema (equivalent to createRecommendationsSchemaDocument in Java)
	schema := buildTemplateRecommenderSchema()

	toolSpec := types.ToolSpecification{
		Name: aws.String("templateRecommender"),
		Description: aws.String(
			"You are a chaos engineer expert, you must provide the most suited template for the reliability issue given to you. " +
				"Given {issue, templates}, return a template recommendation into the field 'template', you should also explain clearly what we can learn with this template in the 'what you can learn from this experiment' field and also provide the next steps to pursue this chaos engineering journey from this template in the field 'what are the next steps', sort the next steps by feasibility and explain why you should start with the first one then proceed to others. It must represent a chaos engineering journey.",
		),
		InputSchema: &types.ToolInputSchemaMemberJson{
			Value: document.NewLazyDocument(schema),
		},
	}

	toolConfig := &types.ToolConfiguration{
		Tools: []types.Tool{
			&types.ToolMemberToolSpec{
				Value: toolSpec,
			},
		},
	}

	// System instructions to force tool usage
	systemPrompt := "You are a reliability assessor and Steadybit expert. You MUST call the tool named 'templateRecommender'."
	systemBlocks := []types.SystemContentBlock{
		&types.SystemContentBlockMemberText{Value: systemPrompt},
	}

	// Build user payload (issues and templates are expected to be JSON)
	userPayload := fmt.Sprintf(
		`{"issue":%s,"templates":%s}`,
		issues,
		templatesAvailable,
	)

	fmt.Println("Approx input tokens (template recommender):", EstimateTokens(userPayload))

	userMessage := types.Message{
		Role: types.ConversationRoleUser,
		Content: []types.ContentBlock{
			&types.ContentBlockMemberText{
				Value: userPayload,
			},
		},
	}

	// Reasoning config (thinking)
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
		return "", fmt.Errorf("bedrock Converse error (templateRecommender): %w", err)
	}

	msgOut, ok := resp.Output.(*types.ConverseOutputMemberMessage)
	if !ok || msgOut == nil {
		return "", fmt.Errorf("unexpected Converse output type for templateRecommender")
	}

	// Prefer tool-use JSON if present, else fall back to any text response
	var toolUseJSON []byte
	var textFallback string

	for _, cb := range msgOut.Value.Content {
		switch v := cb.(type) {
		case *types.ContentBlockMemberToolUse:
			// We only care about the templateRecommender tool
			if v.Value.Input != nil && (v.Value.Name == nil || *v.Value.Name == "templateRecommender") {
				toolUseJSON, err = v.Value.Input.MarshalSmithyDocument()
				if err != nil {
					return "", fmt.Errorf("unable to marshal templateRecommender tool use output: %w", err)
				}
			}
		case *types.ContentBlockMemberText:
			textFallback += v.Value
		}
	}

	var rawOut string
	if len(toolUseJSON) > 0 {
		// Pretty-print JSON if possible (same behavior as FindReliabilityIssues)
		var prettyObj any
		var prettyPrintOut []byte
		if err := json.Unmarshal(toolUseJSON, &prettyObj); err == nil {
			prettyPrintOut, _ = json.MarshalIndent(prettyObj, "", "  ")
			rawOut = string(prettyPrintOut)
			fmt.Println("Template recommender pretty JSON response:\n" + rawOut)
			fmt.Println("Approx output tokens (template recommender):", EstimateTokens(rawOut))
		} else {
			// If pretty-print fails, just return raw JSON
			rawOut = string(toolUseJSON)
		}
	} else {
		// No tool JSON; return streamed text (unlikely if tool usage is enforced, but safe)
		rawOut = textFallback
	}

	if rawOut == "" {
		return "", fmt.Errorf("templateRecommender returned no content")
	}

	return rawOut, nil
}

func buildTemplateRecommenderSchema() map[string]any {
	// issue item schema
	issueProps := map[string]any{
		"title": map[string]any{
			"type": "string",
		},
		"what you can learn from this experiment": map[string]any{
			"type": "string",
		},
		"template": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
		"what are the next steps": map[string]any{
			"type": "string",
		},
	}

	issueRequired := []string{
		"title",
		"what you can learn from this experiment",
		"template",
		"what are the next steps",
	}

	issueItem := map[string]any{
		"type":                 "object",
		"properties":           issueProps,
		"required":             issueRequired,
		"additionalProperties": false,
	}

	// recommendations array schema
	recommendationsArray := map[string]any{
		"type":     "array",
		"maxItems": 3,
		"items":    issueItem,
	}

	// root object schema
	rootProps := map[string]any{
		"recommendations": recommendationsArray,
	}

	rootRequired := []string{"recommendations"}

	return map[string]any{
		"type":                 "object",
		"properties":           rootProps,
		"required":             rootRequired,
		"additionalProperties": false,
	}
}
