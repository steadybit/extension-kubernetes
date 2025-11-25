/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
)

type SingleReliabilityIssueRecord struct {
	Key         string
	WorkloadKey string
	Index       int
	Title       string
	Category    string
	Severity    string
	Priority    string
	Description string
	Raw         string
	Timestamp   time.Time
}

var MaxStoreSize = 50

var singleReliabilityIssuesStore = struct {
	mu    sync.RWMutex
	items map[string]SingleReliabilityIssueRecord
}{
	items: make(map[string]SingleReliabilityIssueRecord),
}

func storeSingleReliabilityIssues(workloadKey, rawJSON string, t time.Time) {
	var root map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &root); err != nil {
		return
	}

	issuesVal, ok := root["issues"]
	if !ok {
		return
	}

	issues, ok := issuesVal.([]interface{})
	if !ok {
		return
	}

	singleReliabilityIssuesStore.mu.Lock()
	defer singleReliabilityIssuesStore.mu.Unlock()

	for idx, item := range issues {
		issue, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		title := getString(issue, "title")
		if title == "" {
			title = fmt.Sprintf("Issue %d", idx+1)
		}

		category := getString(issue, "category")
		severity := getNumber(issue, "severity")
		priority := getNumber(issue, "priority")
		description := getString(issue, "description")

		rawIssueBytes, err := json.Marshal(issue)
		if err != nil {
			rawIssueBytes = nil
		}
		rawIssue := string(rawIssueBytes)

		key := fmt.Sprintf("%s#%d", workloadKey, idx)
		if len(singleReliabilityIssuesStore.items) >= MaxStoreSize {
			for k := range singleReliabilityIssuesStore.items {
				delete(singleReliabilityIssuesStore.items, k)
				break
			}
		}
		singleReliabilityIssuesStore.items[key] = SingleReliabilityIssueRecord{
			Key:         key,
			WorkloadKey: workloadKey,
			Index:       idx,
			Title:       title,
			Category:    category,
			Severity:    severity,
			Priority:    priority,
			Description: description,
			Raw:         rawIssue,
			Timestamp:   t,
		}
	}
}

var (
	_ action_kit_sdk.Action[ReliabilityCheckState] = (*reliabilityCheckAction)(nil)
)

const (
	DeploymentTargetType = "com.steadybit.extension_kubernetes.kubernetes-deployment"
)

type ReliabilityCheckState struct {
	Platform   string
	Kind       string
	Manifest   string
	Result     string
	IsFinished bool
	Key        string
}

type ReliabilityCheckConfig struct {
	Platform string `json:"platform"`
	Kind     string `json:"kind"`
	Manifest string `json:"manifest"`
}

type reliabilityCheckAction struct {
	converse ConverseWrapper
}

func NewReliabilityCheckAction(converse ConverseWrapper) action_kit_sdk.Action[ReliabilityCheckState] {
	return &reliabilityCheckAction{converse: converse}
}

func (a *reliabilityCheckAction) NewEmptyState() ReliabilityCheckState {
	return ReliabilityCheckState{}
}

func (a *reliabilityCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          "com.steadybit.extension_kubernetes.ai.check-reliability-issues",
		Label:       "Check Issues with AI",
		Description: "Uses an AI model to analyze a manifest for reliability issues.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(""),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: DeploymentTargetType,
		}),
		Technology: extutil.Ptr("AI"),
		Category:   extutil.Ptr("Reliability"),

		// This is a Check action, executed once (no status/stop).
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.MarkdownWidget{
				Type:        action_kit_api.ComSteadybitWidgetMarkdown,
				Title:       "Reliability Issues",
				MessageType: "ReliabilityIssues",
				Append:      true,
			},
		}),
	}
}

// Prepare is called before the action is started.
// It validates and copies the config into the state.
func (a *reliabilityCheckAction) Prepare(ctx context.Context, state *ReliabilityCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {

	var cfg ReliabilityCheckConfig
	if err := extconversion.Convert(request.Config, &cfg); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal reliability check config.", err)
	}

	state.Platform = cfg.Platform
	state.Kind = "deployment"

	// Read cluster name from target attributes.
	cluster, ok := firstAttributeValue(request.Target.Attributes, "k8s.cluster-name")
	if !ok {
		return nil, extension_kit.ToError("Missing k8s.cluster-name on target for reliability check.", nil)
	}

	// Derive namespace and name from target attributes.
	namespace, ok := firstAttributeValue(request.Target.Attributes, "k8s.namespace")
	if !ok {
		return nil, extension_kit.ToError("Missing k8s.namespace on target for reliability check.", nil)
	}

	name, ok := firstAttributeValue(request.Target.Attributes, "k8s.name", "k8s.deployment")
	if !ok {
		return nil, extension_kit.ToError("Missing Kubernetes resource name on target for reliability check.", nil)
	}

	// Create Kubernetes client and fetch a sanitized JSON representation of the workload.
	k8sClient, err := NewKubernetesClient()
	if err != nil {
		return nil, extension_kit.ToError("Failed to create Kubernetes client for reliability check.", err)
	}

	manifestJSON, err := GetWorkloadJSON(ctx, k8sClient, state.Kind, namespace, name)
	if err != nil {
		return nil, extension_kit.ToError("Failed to fetch workload manifest for reliability check.", err)
	}
	state.Manifest = manifestJSON
	state.Key = fmt.Sprintf("%s/%s/%s/%s", cluster, namespace, state.Kind, name)
	state.IsFinished = false

	return &action_kit_api.PrepareResult{
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("Prepared reliability check for platform=%s kind=%s", state.Platform, state.Kind),
			},
		}),
	}, nil
}

// Start is called to execute the check once.
// It calls ConverseWrapper.FindReliabilityIssues and returns the result as messages.
func (a *reliabilityCheckAction) Start(ctx context.Context, state *ReliabilityCheckState) (*action_kit_api.StartResult, error) {
	result, err := a.converse.FindReliabilityIssues(
		ctx,
		state.Platform,
		state.Kind,
		state.Manifest,
		nil, // adjust if you add extra options/constraints
	)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find reliability issues via Converse.", err)
	}

	state.Result = result
	state.IsFinished = true
	t := time.Now()

	md := ReliabilityIssuesToMarkdown(result)

	storeSingleReliabilityIssues(state.Key, result, t)

	return &action_kit_api.StartResult{
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Message:   "### Reliability Analysis Summary\n\nThe Kubernetes workload manifest was analyzed using an AI-based reliability assessment.  \nThe model reviewed configuration elements such as resource allocation, workload structure, and operational patterns to identify potential risks or misconfigurations.\n\nPlease note that these findings are generated by an AI model and should be interpreted with appropriate caution.  \nThe reported *severity* and *priority* levels reflect the model’s evaluation from a reliability and chaos-engineering perspective and may not always align perfectly with your environment, operational context, or SLOs.\n\nBelow is a structured summary of the issues identified during the analysis.",
				Type:      extutil.Ptr("ReliabilityIssues"),
				Timestamp: &t,
			},
			{
				Message:   md,
				Type:      extutil.Ptr("ReliabilityIssues"),
				Timestamp: &t,
			},
		}),
	}, nil
}

func (a *reliabilityCheckAction) Status(ctx context.Context, state *ReliabilityCheckState) (*action_kit_api.StatusResult, error) {
	if !state.IsFinished {
		return &action_kit_api.StatusResult{
			Completed: false,
			Messages: extutil.Ptr([]action_kit_api.Message{
				{
					Level:   extutil.Ptr(action_kit_api.Info),
					Message: "Reliability check still running...",
				},
			}),
		}, nil
	}

	return &action_kit_api.StatusResult{
		Completed: true,
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: "Reliability check completed.",
			},
		}),
	}, nil
}

// firstAttributeValue returns the first non-empty value found for the given keys
// in the Steadybit target attributes map.
func firstAttributeValue(attrs map[string][]string, keys ...string) (string, bool) {
	for _, k := range keys {
		if values, ok := attrs[k]; ok && len(values) > 0 && values[0] != "" {
			return values[0], true
		}
	}
	return "", false
}

func ReliabilityIssuesToMarkdown(rawJSON string) string {
	// Optional: unescape HTML entities like &#x27; → '
	rawJSON = html.UnescapeString(rawJSON)

	var root map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &root); err != nil {
		return "The response from the AI is not well formatted, please retry."
	}

	issuesVal, ok := root["issues"]
	if !ok {
		// fallback: just dump as code block
		return fmt.Sprintf("```json\n%s\n```", rawJSON)
	}

	issues, ok := issuesVal.([]interface{})
	if !ok {
		return fmt.Sprintf("```json\n%s\n```", rawJSON)
	}

	var b strings.Builder
	b.WriteString("### Detected Issues\n\n")

	for i, item := range issues {
		issue, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		title := getString(issue, "title")
		if title == "" {
			title = fmt.Sprintf("Issue %d", i+1)
		}

		severity := getNumber(issue, "severity")
		priority := getNumber(issue, "priority")
		category := getString(issue, "category")
		description := getString(issue, "description")

		// Issue heading
		b.WriteString(fmt.Sprintf("#### %d. %s\n\n", i+1, title))

		// Meta info as a small table
		b.WriteString("| Field    | Value |\n")
		b.WriteString("|----------|-------|\n")
		if category != "" {
			b.WriteString(fmt.Sprintf("| Category | %s |\n", category))
		}
		if severity != "" {
			b.WriteString(fmt.Sprintf("| Severity | %s |\n", severity))
		}
		if priority != "" {
			b.WriteString(fmt.Sprintf("| Priority | %s |\n", priority))
		}
		b.WriteString("\n")

		// Description
		if description != "" {
			b.WriteString(description)
			b.WriteString("  ")
		}

		// Lists
		writeStringList(&b, "Signals", issue["signals"])
		writeStringList(&b, "Fixes", issue["fixes"])
		writeStringList(&b, "Experiments", issue["experiments"])

		// Target types
		writeStringList(&b, "Target types", issue["target_types"])

		b.WriteString("\n")
	}

	return b.String()
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getNumber(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			// JSON numbers come out as float64
			return strconv.FormatFloat(n, 'f', -1, 64)
		case int:
			return strconv.Itoa(n)
		}
	}
	return ""
}

func writeStringList(b *strings.Builder, title string, raw interface{}) {
	items, ok := raw.([]interface{})
	if !ok || len(items) == 0 {
		return
	}

	b.WriteString(fmt.Sprintf("**%s**:\n", title))
	for _, it := range items {
		if s, ok := it.(string); ok {
			b.WriteString(fmt.Sprintf("- %s\n", s))
		}
	}
	b.WriteString("\n")
}
