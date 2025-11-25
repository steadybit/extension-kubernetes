package ai

// GetReliabilityIssuesTool returns the tool definition for the reliability issues tool.
func GetReliabilityIssuesTool() ToolDefinition {
	return ReliabilityIssuesTool
}

// GetTemplateRecommenderTool returns the tool definition for the template recommender tool.
func GetTemplateRecommenderTool() ToolDefinition {
	return TemplateRecommenderTool
}

var ReliabilityIssuesTool = ToolDefinition{
	Name: "reliabilityIssues",
	Description: "Returns the reliability issues about the given resources. " +
		"Include AT MOST 3 issues and choose the most critical ones. " +
		"Given {technology, target_types, resource_json, optional reliability_priority}, " +
		"return reliability issues ordered by 'priority' where 1=highest. " +
		"If reliability_priority is provided, rank by it first; otherwise rank by criticality. " +
		"The target_types just need to be passed in the output.",
	SchemaJSON: ReliabilityIssuesSchemaJSON,
}

// TemplateRecommenderTool is the Go equivalent of your Java createToolConfig().
var TemplateRecommenderTool = ToolDefinition{
	Name: "templateRecommender",
	Description: "You are a chaos engineer expert, you must provide the most suited template for the reliability issue given to you. " +
		"Given {issue, templates}, return a template recommendation into the field 'template'. " +
		"You must also explain clearly what we can learn from this experiment in the field 'what you can learn from this experiment', " +
		"and you must provide the next steps to pursue this chaos engineering journey in the field 'what are the next steps'. " +
		"Sort the next steps by feasibility and explain why the first one should be done before the others. " +
		"It must represent a chaos engineering journey.",
	SchemaJSON: TemplateRecommenderSchemaJSON,
}

// JSON schema translated from createRecommendationsSchemaDocument()
const TemplateRecommenderSchemaJSON = `{
  "type": "object",
  "properties": {
    "recommendations": {
      "type": "array",
      "maxItems": 3,
      "items": {
        "type": "object",
        "properties": {
          "title": {
            "type": "string"
          },
          "what you can learn from this experiment": {
            "type": "string"
          },
          "template": {
            "type": "array",
            "items": { "type": "string" }
          },
          "what are the next steps": {
            "type": "string"
          }
        },
        "required": [
          "title",
          "what you can learn from this experiment",
          "template",
          "what are the next steps"
        ],
        "additionalProperties": false
      }
    }
  },
  "required": ["recommendations"],
  "additionalProperties": false
}`

// JSON schema translated from createIssuesSchemaDocument()
const ReliabilityIssuesSchemaJSON = `{
  "type": "object",
  "properties": {
    "issues": {
      "type": "array",
      "maxItems": 3,
      "items": {
        "type": "object",
        "properties": {
          "title":       { "type": "string" },
          "description": { "type": "string" },
          "category":    { "type": "string" },
          "priority":    { "type": "integer", "minimum": 1 },
          "severity":    { "type": "integer", "minimum": 1, "maximum": 10 },
          "signals": {
            "type": "array",
            "items": { "type": "string" }
          },
          "fixes": {
            "type": "array",
            "items": { "type": "string" }
          },
          "experiments": {
            "type": "array",
            "items": { "type": "string" }
          },
          "target_types": {
            "type": "array",
            "items": { "type": "string" }
          }
        },
        "required": [
          "title",
          "description",
          "category",
          "priority",
          "severity",
          "signals",
          "fixes",
          "experiments",
          "target_types"
        ],
        "additionalProperties": false
      }
    }
  },
  "required": ["issues"],
  "additionalProperties": false
}`
