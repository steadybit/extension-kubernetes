package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"html"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

type reliabilityCheckDeploymentAction struct {
	converse ConverseWrapper
}

type reliabilityCheckStatefulSetAction struct {
	converse ConverseWrapper
}

type ReliabilityCheckState struct {
	Technology  string
	Namespace   string
	Name        string
	ClusterName string
	Kind        string
	Manifest    string
	Result      string
	IsFinished  bool
	Key         string
	JobID       string
}

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

const (
	targetIcon = "data:image/svg+xml,%3Csvg%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22currentColor%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.7998%202C11.3298%202%2011.7597%202.42998%2011.7598%202.95996C11.7598%203.48996%2011.3298%203.91992%2010.7998%203.91992C7.00996%203.92003%203.92003%206.99996%203.91992%2010.7998C3.91992%2014.5997%206.99989%2017.6903%2010.7998%2017.6904C14.5998%2017.6904%2017.6904%2014.5998%2017.6904%2010.7998C17.6905%2010.2699%2018.1205%209.83984%2018.6504%209.83984C19.1801%209.84006%2019.6102%2010.27%2019.6104%2010.7998C19.6104%2012.8696%2018.8902%2014.7696%2017.6904%2016.2695L22.21%2020.79C22.6%2021.18%2022.6%2021.8102%2022.21%2022.2002C22.01%2022.3901%2021.7599%2022.4902%2021.5%2022.4902C21.2401%2022.4902%2020.99%2022.4001%2020.79%2022.2002L16.2695%2017.6797C14.7696%2018.8796%2012.8697%2019.5996%2010.7998%2019.5996C5.94989%2019.5995%202%2015.6497%202%2010.7998C2.00011%205.94996%205.94996%202.00011%2010.7998%202ZM10.2598%205.98047C10.2998%205.84047%2010.4903%205.84047%2010.5303%205.98047L11.4805%209.80957C11.4905%209.85957%2011.5801%209.91016%2011.5801%209.91016L15.4102%2010.8604C15.5499%2010.9005%2015.5499%2011.0898%2015.4102%2011.1299L11.5801%2012.0801C11.5321%2012.0897%2011.4843%2012.1728%2011.4805%2012.1797L10.5303%2016.0098C10.4903%2016.1498%2010.2998%2016.1498%2010.2598%2016.0098L9.30957%2012.1797C9.29922%2012.13%209.21065%2012.0805%209.20996%2012.0801L5.37988%2011.1299C5.24008%2011.0898%205.24012%2010.9005%205.37988%2010.8604L9.20996%209.91016C9.25996%209.90016%209.30957%209.80957%209.30957%209.80957L10.2598%205.98047ZM15.9902%202.24023C16.0302%202.10023%2016.2198%202.10023%2016.2598%202.24023L16.9404%204.95996C16.9427%204.96403%2016.9912%205.04977%2017.04%205.05957L19.7598%205.74023C19.8998%205.78023%2019.8998%205.96977%2019.7598%206.00977L17.04%206.69043C17.036%206.69268%2016.9503%206.74122%2016.9404%206.79004L16.2598%209.50977C16.2198%209.64977%2016.0302%209.64977%2015.9902%209.50977L15.3096%206.79004C15.3072%206.78585%2015.2587%206.70023%2015.21%206.69043L12.4902%206.00977C12.3502%205.97977%2012.3502%205.78023%2012.4902%205.74023L15.21%205.05957C15.2142%205.05718%2015.2998%205.00871%2015.3096%204.95996L15.9902%202.24023Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3C%2Fsvg%3E%0A"
)

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

	b.WriteString(fmt.Sprintf("#### %s:\n", title))
	for _, it := range items {
		if s, ok := it.(string); ok {
			b.WriteString(fmt.Sprintf("- %s\n", s))
		}
	}
	b.WriteString("\n")
}

func sparklePulse() string {
	sparkles := []string{
		"⋆˙⟡",
		"⋆✴︎˚｡⋆",
		"✦⋆｡˚",
	}
	return sparkles[rnd.Intn(len(sparkles))]
}

func ReliabilityIssuesToMarkdown(rawJSON string) string {
	// Optional: unescape HTML entities like &#x27; → '
	rawJSON = html.UnescapeString(rawJSON)

	var root map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &root); err != nil {
		return "The response from the AI is not well formatted, please retry.\n"
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
			b.WriteString(description + "\n\n")
		}
		writeStringList(&b, "Fixes", issue["fixes"])
		writeStringList(&b, "Experiments", issue["experiments"])
	}

	return b.String()
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

type ReliabilityJob struct {
	Done      bool
	Result    string
	Err       error
	Timestamp time.Time
}

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

var reliabilityJobs = struct {
	mu sync.RWMutex
	m  map[string]*ReliabilityJob
}{
	m: make(map[string]*ReliabilityJob),
}

var (
	MaxStoreSize                 = 50
	rnd                          = rand.New(rand.NewSource(time.Now().UnixNano()))
	singleReliabilityIssuesStore = struct {
		mu    sync.RWMutex
		items map[string]SingleReliabilityIssueRecord
	}{
		items: make(map[string]SingleReliabilityIssueRecord),
	}
)

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

func prepare(ctx context.Context, state *ReliabilityCheckState, request action_kit_api.PrepareActionRequestBody) error {
	// Read cluster name from target attributes.
	var ok bool
	state.ClusterName, ok = firstAttributeValue(request.Target.Attributes, "k8s.cluster-name")
	if !ok {
		return extension_kit.ToError("Missing k8s.cluster-name on target for reliability check.", nil)
	}

	// Derive namespace and name from target attributes.
	state.Namespace, ok = firstAttributeValue(request.Target.Attributes, "k8s.namespace")
	if !ok {
		return extension_kit.ToError("Missing k8s.namespace on target for reliability check.", nil)
	}

	state.Name, ok = firstAttributeValue(request.Target.Attributes, "k8s."+state.Kind)
	if !ok {
		return extension_kit.ToError("Missing Kubernetes resource name on target for reliability check.", nil)
	}

	// Create Kubernetes client and fetch a sanitized JSON representation of the workload.
	k8sClient, err := NewKubernetesClient()
	if err != nil {
		return extension_kit.ToError("Failed to create Kubernetes client for reliability check.", err)
	}

	manifestJSON, err := GetWorkloadJSON(ctx, k8sClient, state.Kind, state.Namespace, state.Name)
	if err != nil {
		return extension_kit.ToError("Failed to fetch workload manifest for reliability check.", err)
	}
	state.Technology = "kubernetes"
	state.Manifest = manifestJSON
	state.Key = fmt.Sprintf("%s/%s/%s/%s", state.ClusterName, state.Namespace, state.Kind, state.Name)
	state.IsFinished = false

	return nil
}

func status(state *ReliabilityCheckState) (*action_kit_api.StatusResult, error) {
	if state.JobID == "" {
		return nil, extension_kit.ToError("Status called before Start (missing job ID).", nil)
	}

	reliabilityJobs.mu.RLock()
	job, ok := reliabilityJobs.m[state.JobID]
	reliabilityJobs.mu.RUnlock()

	if !ok {
		return nil, extension_kit.ToError(fmt.Sprintf("Unknown job ID: %s", state.JobID), nil)
	}

	// Job finished, check for errors first
	if job.Err != nil {
		t := job.Timestamp

		// Cleanup completed job from the map
		reliabilityJobs.mu.Lock()
		delete(reliabilityJobs.m, state.JobID)
		reliabilityJobs.mu.Unlock()

		errMsg := fmt.Sprintf("AI reliability analysis failed: %v \n", job.Err)
		return &action_kit_api.StatusResult{
			Completed: true,
			Messages: extutil.Ptr([]action_kit_api.Message{
				{
					Level:     extutil.Ptr(action_kit_api.Error),
					Type:      extutil.Ptr("ReliabilityIssues"),
					Message:   errMsg,
					Timestamp: &t,
				},
			}),
		}, nil
	}

	if !job.Done {
		return &action_kit_api.StatusResult{
			Completed: false,
			Messages: extutil.Ptr([]action_kit_api.Message{
				{
					Level:   extutil.Ptr(action_kit_api.Info),
					Type:    extutil.Ptr("ReliabilityIssues"),
					Message: sparklePulse(),
				},
			}),
		}, nil
	}

	state.Result = job.Result
	state.IsFinished = true
	t := job.Timestamp

	md := ReliabilityIssuesToMarkdown(job.Result)
	storeSingleReliabilityIssues(state.Key, job.Result, t)
	// Cleanup completed job from the map
	reliabilityJobs.mu.Lock()
	delete(reliabilityJobs.m, state.JobID)
	reliabilityJobs.mu.Unlock()

	return &action_kit_api.StatusResult{
		Completed: true,
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Message:   "\n",
				Timestamp: &t,
				Type:      extutil.Ptr("ReliabilityIssues"),
			},
			{
				Message:   "---",
				Timestamp: &t,
				Type:      extutil.Ptr("ReliabilityIssues"),
			},
			{
				Message:   "# Reliability Analysis Summary\n\n\n  The Kubernetes workload manifest was analyzed using an AI-based reliability assessment, for token consumption we limit to 3 issues.  \nThe model reviewed configuration elements such as resource allocation, workload structure, and operational patterns to identify potential risks or misconfigurations.\n\nPlease note that these findings are generated by an AI model and should be interpreted with appropriate caution.  \nThe reported *severity* and *priority* levels reflect the model’s evaluation from a reliability and chaos-engineering perspective and may not always align perfectly with your environment, operational context, or SLOs.\n\nBelow is a structured summary of the issues identified during the analysis.",
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
