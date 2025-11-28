package ai

import (
	"context"
	"encoding/json"
	"fmt"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"strings"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

var (
	_ action_kit_sdk.Action[AnalysisState] = (*templateRecommendationAction)(nil)
)

// Action implementation
type templateRecommendationAction struct {
	converse        ConverseWrapper
	templatesClient *TemplateAPIClient
}

func NewTemplateRecommendationAction(converse ConverseWrapper, templatesClient *TemplateAPIClient) action_kit_sdk.Action[AnalysisState] {
	return &templateRecommendationAction{converse: converse, templatesClient: templatesClient}
}

func (a *templateRecommendationAction) NewEmptyState() AnalysisState {
	return AnalysisState{}
}

func (a *templateRecommendationAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          "com.steadybit.extension_kubernetes.ai.templates.recommend",
		Label:       "Recommend Templates From Target",
		Description: "Uses an AI model to recommend chaos templates for a given AI-found reliability issue.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(targetIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          ReliabilityIssueTargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.QuantityRestrictionExactlyOne),
		}),
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
	}
}

// Prepare reads the issue and candidate templates from the target attributes.
func (a *templateRecommendationAction) Prepare(
	ctx context.Context,
	state *AnalysisState,
	request action_kit_api.PrepareActionRequestBody,
) (*action_kit_api.PrepareResult, error) {
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

	issueJSON := IssueRecordToJSON(rec)
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
func (a *templateRecommendationAction) Start(
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

func (a *templateRecommendationAction) Status(ctx context.Context, state *AnalysisState) (*action_kit_api.StatusResult, error) {
	return status(state, "TemplateRecommendations", TemplateRecommendationsToMarkdown)
}

type recommendation struct {
	Title           string   `json:"title"`
	WhatYouCanLearn string   `json:"what you can learn from this experiment"`
	Template        []string `json:"template"`
	NextSteps       string   `json:"what are the next steps"`
}

type recommendationResult struct {
	Recommendations []recommendation `json:"recommendations"`
}

func TemplateRecommendationsToMarkdown(raw string) string {
	var parsed recommendationResult
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil || len(parsed.Recommendations) == 0 {
		// Fallback: show raw output if JSON parsing fails
		return "```\n" + raw + "\n```"
	}

	md := "# Template Recommendation Summary\n\n" +
		"The selected reliability issue was analyzed using an AI-based template recommender.  \n" +
		"The model reviewed the issue context (including signals, experiments, and potential fixes where available) and matched it with suitable chaos experiment templates.\n\n" +
		"Please note that these recommendations are generated by an AI model and should be interpreted with appropriate caution.  \n" +
		"Use them as a guided starting point for your chaos engineering journey rather than definitive prescriptions.\n\n" +
		"Below is a structured summary of the recommended templates and how they relate to your issue.\n\n"
	for i, r := range parsed.Recommendations {
		md += fmt.Sprintf("### %d. %s\n\n", i+1, r.Title)
		if len(r.Template) > 0 {
			md += "#### Suggested template(s):\n\n"
			for _, t := range r.Template {
				if extconfig.Config.PlatformBaseURL != "" {
					// Produce a clickable link to the template inside the platform
					link := fmt.Sprintf("%s/settings/templates;selectedTemplateId=%s~", extconfig.Config.PlatformBaseURL, t)
					md += fmt.Sprintf("- [`%s`](%s)\n", t, link)
				} else {
					// Fallback if PLATFORM_API_HOST is not defined
					md += fmt.Sprintf("- `%s`\n", t)
				}
			}
			md += "\n"
		}
		if r.WhatYouCanLearn != "" {
			md += "#### What you can learn from this experiment:\n\n"
			md += r.WhatYouCanLearn + "\n\n"
		}
		if r.NextSteps != "" {
			md += "#### What are the next steps:\n\n"

			// Split by newline or by numbered/bulleted entries from the model
			steps := strings.Split(r.NextSteps, "\n")
			for _, raw := range steps {
				s := strings.TrimSpace(raw)
				if s == "" {
					continue
				}
				md += "- " + s + "\n"
			}

			md += "\n"
		}
		md += "---\n\n"
	}
	return md
}
