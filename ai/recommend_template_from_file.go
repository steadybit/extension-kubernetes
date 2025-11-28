package ai

import (
	"context"
	"fmt"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extconversion"
	"os"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

var (
	_ action_kit_sdk.Action[AnalysisState] = (*templateRecommendationFromFileAction)(nil)
)

type templateRecommendationFromFileAction struct {
	converse        ConverseWrapper
	templatesClient *TemplateAPIClient
}

type TemplateRecommendationConfig struct {
	IssueFile string `json:"issueFile"`
}

func NewTemplateRecommendationFromFileAction(converse ConverseWrapper, templatesClient *TemplateAPIClient) action_kit_sdk.Action[AnalysisState] {
	return &templateRecommendationFromFileAction{converse: converse, templatesClient: templatesClient}
}

func (a *templateRecommendationFromFileAction) NewEmptyState() AnalysisState {
	return AnalysisState{}
}

func (a *templateRecommendationFromFileAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          "com.steadybit.extension_kubernetes.ai.templates.recommend-from-file",
		Label:       "Recommend Templates From File",
		Description: "Uses an AI model to recommend chaos templates for a given AI-found reliability issue.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(targetIcon),
		Technology:  extutil.Ptr("AI"),
		Category:    extutil.Ptr("Reliability"),
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.MarkdownWidget{
				Type:        action_kit_api.ComSteadybitWidgetMarkdown,
				Title:       "Template Recommendations",
				MessageType: "TemplateRecommendations",
				Append:      true,
			},
		}),
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("3s"),
		}),
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:        "issueFile",
				Label:       "Issue JSON",
				Description: extutil.Ptr("JSON file describing the issue."),
				Type:        action_kit_api.ActionParameterTypeFile,
				Required:    extutil.Ptr(true),
				AcceptedFileTypes: extutil.Ptr([]string{
					".json",
				}),
			},
		},
	}
}

// Prepare reads the issue and candidate templates from the target attributes.
func (a *templateRecommendationFromFileAction) Prepare(ctx context.Context, state *AnalysisState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var cfg TemplateRecommendationConfig
	if err := extconversion.Convert(request.Config, &cfg); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal template recommendation config.", err)
	}

	var issueJSON string

	if cfg.IssueFile != "" {
		// If an issue JSON file has been provided, read and use it directly.
		bytes, err := os.ReadFile(cfg.IssueFile)
		if err != nil {
			return nil, extension_kit.ToError("Failed to read issue JSON file.", err)
		}
		issueJSON = string(bytes)
	} else {
		// Fall back to constructing the issue from the AI issue target attributes.
		attrs := request.Target.Attributes

		title, _ := firstAttributeValue(attrs, "k8s.ai.reliability_issues.title")
		category, _ := firstAttributeValue(attrs, "k8s.ai.reliability_issues.category")
		severity, _ := firstAttributeValue(attrs, "k8s.ai.reliability_issues.severity")
		priority, _ := firstAttributeValue(attrs, "k8s.ai.reliability_issues.priority")
		description, _ := firstAttributeValue(attrs, "k8s.ai.reliability_issues.description")

		rec := SingleReliabilityIssueRecord{
			Title:       title,
			Category:    category,
			Severity:    severity,
			Priority:    priority,
			Description: description,
			Signals:     attrs["k8s.ai.reliability_issues.signals"],
			Experiments: attrs["k8s.ai.reliability_issues.experiments"],
			Fixes:       attrs["k8s.ai.reliability_issues.fixes"],
		}

		issueJSON = IssueRecordToJSON(rec)
	}

	state.IssueJSON = issueJSON

	if a.templatesClient == nil {
		return nil, extension_kit.ToError("Template API client is not configured for template recommendation.", nil)
	}

	// Fetch all templates for Kubernetes deployments from the Steadybit platform API.
	rawTemplates, err := a.templatesClient.FetchTemplates("com.steadybit.extension_kubernetes.kubernetes-deployment")
	if err != nil {
		return nil, extension_kit.ToError("Failed to fetch templates for AI recommendation.", err)
	}

	// Build a compact, ranked JSON list of templates suitable as LLM context.
	compactTemplates, err := BuildTemplatesAvailableJSON(issueJSON, "deployment", rawTemplates, 10)
	if err != nil {
		return nil, extension_kit.ToError("Failed to build compact templates JSON for AI recommendation.", err)
	}

	state.TemplatesJSON = compactTemplates

	return &action_kit_api.PrepareResult{
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("TemplateRecommendations"),
				Message: "# AI Template Recommendation",
			},
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("TemplateRecommendations"),
				Message: "---",
			},
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("TemplateRecommendations"),
				Message: "## Preparation\nAI issue and candidate templates retrieved. Starting recommendation...",
			},
		}),
	}, nil
}

// Start triggers the template recommendation asynchronously using the shared job store.
func (a *templateRecommendationFromFileAction) Start(
	ctx context.Context,
	state *AnalysisState,
) (*action_kit_api.StartResult, error) {
	jobID := fmt.Sprintf("job-%d", time.Now().UnixNano())
	state.JobID = jobID

	reliabilityJobs.mu.Lock()
	reliabilityJobs.m[jobID] = &ReliabilityJob{Done: false}
	reliabilityJobs.mu.Unlock()

	go func(issueJSON, templatesJSON, jobID string) {
		result, err := a.converse.RecommendTemplate(issueJSON, templatesJSON)

		reliabilityJobs.mu.Lock()
		defer reliabilityJobs.mu.Unlock()
		job := reliabilityJobs.m[jobID]
		job.Done = true
		job.Result = result
		job.Err = err
		job.Timestamp = time.Now()
	}(state.IssueJSON, state.TemplatesJSON, jobID)

	return &action_kit_api.StartResult{
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("TemplateRecommendations"),
				Message: "### Analyzing issue and available templates ✧˖°.",
			},
		}),
	}, nil
}

func (a *templateRecommendationFromFileAction) Status(ctx context.Context, state *AnalysisState) (*action_kit_api.StatusResult, error) {
	return status(state, "TemplateRecommendations", TemplateRecommendationsToMarkdown)
}

func (a *templateRecommendationFromFileAction) Stop(
	ctx context.Context,
	state *AnalysisState,
) (*action_kit_api.StopResult, error) {
	// No external processes or long-running resources to clean up.
	// File uploads are handled by the platform and are only read once in Prepare.
	return &action_kit_api.StopResult{}, nil
}
