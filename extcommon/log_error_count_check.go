// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LogErrorCountMetricName is the cumulative number of log lines matching the error pattern per pod.
	LogErrorCountMetricName = "k8s_log_error_count"
	// LogLineCountMetricName is the cumulative number of log lines read per pod.
	LogLineCountMetricName = "k8s_log_line_count"

	// DefaultLogErrorPattern matches the usual error markers of plain-text, logfmt and JSON logs
	// (e.g. "ERROR", "level=error", "\"level\":\"error\"", "panic:", "NullPointerException").
	DefaultLogErrorPattern = `(?i)(\b(error|fatal|panic|critical)\b|exception\b)`

	LogErrorCountCheckModeNone           LogErrorCountCheckMode = "none"
	LogErrorCountCheckModeTotalExceeded  LogErrorCountCheckMode = "totalErrorsExceeded"
	LogErrorCountCheckModePerPodExceeded LogErrorCountCheckMode = "podErrorsExceeded"
)

const (
	// Lines longer than this are truncated before matching.
	logMaxLineLength = 64 * 1024
	// Counted lines are flushed to the cursor at the latest after this many lines.
	logFlushLines = 512
	// Log streams are closed at the latest this long after the step should have ended, in case stop is never called.
	logStreamGracePeriod = 30 * time.Second
)

type LogErrorCountCheckMode string

// PodLogStreamer opens a log stream of a single container. Implemented by *client.Client.
type PodLogStreamer interface {
	PodLogs(ctx context.Context, namespace, podName string, opts *corev1.PodLogOptions) (io.ReadCloser, error)
}

type LogErrorCountCheckAction struct {
	Client             *client.Client
	ActionId           string
	TargetType         string
	TargetTypeLabel    string
	SelectionTemplates *action_kit_api.TargetSelectionTemplates
	GetTarget          func(request action_kit_api.PrepareActionRequestBody) string
	// GetPods returns the pods currently belonging to the target. It is evaluated on every status call, so
	// pods created during the step (e.g. by a rollout) are picked up as well.
	GetPods func(k8s *client.Client, namespace string, target string) ([]*corev1.Pod, error)
	// MetricLabelKey is the k8s label key used to identify the workload in emitted metrics (e.g. "k8s.deployment").
	MetricLabelKey string
	// logStreamer overrides the client for reading logs (tests only).
	logStreamer PodLogStreamer
	// waitForReads makes Status wait until all opened log streams have ended (tests only).
	waitForReads bool
}

// ContainerLogCursor remembers how far the logs of a container have been read and what has been counted so far.
type ContainerLogCursor struct {
	Pod       string    `json:"pod"`
	Container string    `json:"container"`
	LastTime  time.Time `json:"lastTime"`
	// SeenAtLast is the number of lines already counted carrying exactly LastTime. The API only supports
	// second precision for sinceTime, so lines of the last second are returned again and must be skipped.
	SeenAtLast int `json:"seenAtLast"`
	Errors     int `json:"errors"`
	Lines      int `json:"lines"`
	// RestartCount is the restart count of the container instance whose logs were followed last. When a stream
	// is opened for a newer instance, the remaining logs of the previous (crashed) instance are read first.
	RestartCount int32 `json:"restartCount"`
}

type LogErrorCountCheckState struct {
	ExecutionId    uuid.UUID              `json:"executionId"`
	Start          time.Time              `json:"start"`
	End            time.Time              `json:"end"`
	Namespace      string                 `json:"namespace"`
	Target         string                 `json:"target"`
	MetricLabelKey string                 `json:"metricLabelKey"`
	ErrorPattern   string                 `json:"errorPattern"`
	IgnorePattern  string                 `json:"ignorePattern"`
	ContainerName  string                 `json:"containerName"`
	CheckMode      LogErrorCountCheckMode `json:"checkMode"`
	MaxErrorCount  int                    `json:"maxErrorCount"`
	FailEarly      bool                   `json:"failEarly"`
	// Cursors is a snapshot of the log streams' progress, taken on every status call. The streams themselves live
	// in memory; if the extension restarts during the step, they are resumed from this snapshot.
	Cursors map[string]*ContainerLogCursor `json:"cursors"`
}

var _ action_kit_sdk.Action[LogErrorCountCheckState] = (*LogErrorCountCheckAction)(nil)
var _ action_kit_sdk.ActionWithStatus[LogErrorCountCheckState] = (*LogErrorCountCheckAction)(nil)
var _ action_kit_sdk.ActionWithStop[LogErrorCountCheckState] = (*LogErrorCountCheckAction)(nil)

func (f LogErrorCountCheckAction) NewEmptyState() LogErrorCountCheckState {
	return LogErrorCountCheckState{}
}

func (f LogErrorCountCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          f.ActionId,
		Label:       f.TargetTypeLabel + " Error Log Count",
		Description: "Counts the error log lines of the " + strings.ToLower(f.TargetTypeLabel) + "'s pods and optionally fails when a threshold is exceeded. Only the counts are reported, the log lines themselves are not collected.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        new("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTUgM0gxNEwxOSA4VjEyIiBzdHJva2U9IiMxRDI2MzIiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIi8+CjxwYXRoIGQ9Ik01IDNWMjFIMTIiIHN0cm9rZT0iIzFEMjYzMiIgc3Ryb2tlLXdpZHRoPSIyIiBzdHJva2UtbGluZWNhcD0icm91bmQiIHN0cm9rZS1saW5lam9pbj0icm91bmQiLz4KPHBhdGggZD0iTTggOUgxMiIgc3Ryb2tlPSIjMUQyNjMyIiBzdHJva2Utd2lkdGg9IjIiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8cGF0aCBkPSJNOCAxM0gxMSIgc3Ryb2tlPSIjMUQyNjMyIiBzdHJva2Utd2lkdGg9IjIiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8Y2lyY2xlIGN4PSIxNy41IiBjeT0iMTcuNSIgcj0iNC41IiBmaWxsPSIjMUQyNjMyIi8+CjxwYXRoIGQ9Ik0xNy41IDE1VjE3LjgiIHN0cm9rZT0id2hpdGUiIHN0cm9rZS13aWR0aD0iMS41IiBzdHJva2UtbGluZWNhcD0icm91bmQiLz4KPGNpcmNsZSBjeD0iMTcuNSIgY3k9IjE5LjkiIHI9IjAuOCIgZmlsbD0id2hpdGUiLz4KPC9zdmc+Cg=="),
		Technology:  new("Kubernetes"),

		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		TargetSelection: new(action_kit_api.TargetSelection{
			TargetType:          f.TargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.QuantityRestrictionAll),
			SelectionTemplates:  f.SelectionTemplates,
		}),
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  new("How long should the error logs be counted."),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: new("60s"),
				Order:        new(1),
				Required:     new(true),
			},
			{
				Name:         "checkMode",
				Label:        "Fail when",
				Description:  new("Whether the check should fail when too many error log lines are counted. Use 'Never' to only visualize the error log count."),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: new(string(LogErrorCountCheckModeNone)),
				Order:        new(2),
				Required:     new(true),
				Options: new([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Never (visualize only)",
						Value: string(LogErrorCountCheckModeNone),
					},
					action_kit_api.ExplicitParameterOption{
						Label: "total error count > max",
						Value: string(LogErrorCountCheckModeTotalExceeded),
					},
					action_kit_api.ExplicitParameterOption{
						Label: "error count of any pod > max",
						Value: string(LogErrorCountCheckModePerPodExceeded),
					},
				}),
			},
			{
				Name:         "maxErrorCount",
				Label:        "Max error count",
				Description:  new("Maximum number of error log lines allowed during the step. Ignored when 'Fail when' is 'Never'."),
				Type:         action_kit_api.ActionParameterTypeInteger,
				DefaultValue: new("0"),
				Order:        new(3),
				Required:     new(true),
			},
			{
				Name:         "errorPattern",
				Label:        "Error pattern",
				Description:  new("Regular expression (RE2 syntax) identifying an error log line. The default pattern is evaluated by an optimized matcher; custom patterns cost considerably more CPU on workloads logging a lot."),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: new(DefaultLogErrorPattern),
				Order:        new(4),
				Required:     new(true),
				Advanced:     new(true),
			},
			{
				Name:        "ignorePattern",
				Label:       "Ignore pattern",
				Description: new("Optional regular expression (RE2 syntax). Lines matching it are never counted as errors."),
				Type:        action_kit_api.ActionParameterTypeString,
				Order:       new(5),
				Required:    new(false),
				Advanced:    new(true),
			},
			{
				Name:        "containerName",
				Label:       "Container name",
				Description: new("Only count the logs of this container. Leave empty to count the logs of all containers (e.g. including sidecars)."),
				Type:        action_kit_api.ActionParameterTypeString,
				Order:       new(6),
				Required:    new(false),
				Advanced:    new(true),
			},
			{
				Name:         "failEarly",
				Label:        "Fail early",
				Description:  new("If enabled, the check fails as soon as the threshold is exceeded. If disabled, the error logs are counted for the whole duration and the check fails at the end of the step."),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: new("false"),
				Order:        new(7),
				Required:     new(false),
				Advanced:     new(true),
			},
		},
		Widgets: new([]action_kit_api.Widget{
			action_kit_api.LineChartWidget{
				Type:  action_kit_api.ComSteadybitWidgetLineChart,
				Title: "Kubernetes Error Log Count",
				Identity: action_kit_api.LineChartWidgetIdentityConfig{
					MetricName: LogErrorCountMetricName,
					From:       "k8s.pod.name",
					Mode:       action_kit_api.ComSteadybitWidgetLineChartIdentityModeSelect,
				},
				Tooltip: new(action_kit_api.LineChartWidgetTooltipConfig{
					MetricValueTitle: new("Error log lines"),
					AdditionalContent: []action_kit_api.LineChartWidgetTooltipContent{
						{From: "k8s.pod.name", Title: "Pod"},
						{From: "k8s.namespace", Title: "Namespace"},
						{From: f.MetricLabelKey, Title: f.TargetTypeLabel},
					},
				}),
			},
		}),
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status: new(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: new("2s"),
		}),
		Stop: new(action_kit_api.MutatingEndpointReference{}),
	}
}

func (f LogErrorCountCheckAction) Prepare(_ context.Context, state *LogErrorCountCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	errorPattern := extutil.ToString(request.Config["errorPattern"])
	if strings.TrimSpace(errorPattern) == "" {
		errorPattern = DefaultLogErrorPattern
	}
	if _, err := regexp.Compile(errorPattern); err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Invalid error pattern '%s'.", errorPattern), err)
	}
	ignorePattern := strings.TrimSpace(extutil.ToString(request.Config["ignorePattern"]))
	if ignorePattern != "" {
		if _, err := regexp.Compile(ignorePattern); err != nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Invalid ignore pattern '%s'.", ignorePattern), err)
		}
	}

	checkMode := LogErrorCountCheckMode(extutil.ToString(request.Config["checkMode"]))
	switch checkMode {
	case "":
		checkMode = LogErrorCountCheckModeNone
	case LogErrorCountCheckModeNone, LogErrorCountCheckModeTotalExceeded, LogErrorCountCheckModePerPodExceeded:
	default:
		return nil, extension_kit.ToError(fmt.Sprintf("Unknown check mode '%s'.", checkMode), nil)
	}

	maxErrorCount := int(extutil.ToInt64(request.Config["maxErrorCount"]))
	if maxErrorCount < 0 {
		return nil, extension_kit.ToError("Max error count must not be negative.", nil)
	}

	namespace := request.Target.Attributes["k8s.namespace"][0]
	target := f.GetTarget(request)
	pods, err := f.GetPods(f.Client, namespace, target)
	if err != nil {
		return nil, err
	}
	containerName := strings.TrimSpace(extutil.ToString(request.Config["containerName"]))
	if err := checkLogStreamLimits(countContainers(pods, containerName)); err != nil {
		return nil, err
	}

	duration := time.Duration(extutil.ToInt64(request.Config["duration"])) * time.Millisecond
	state.Start = time.Now()
	state.End = state.Start.Add(duration)
	state.Namespace = namespace
	state.Target = target
	state.MetricLabelKey = f.MetricLabelKey
	state.ErrorPattern = errorPattern
	state.IgnorePattern = ignorePattern
	state.ExecutionId = request.ExecutionId
	state.ContainerName = containerName
	state.CheckMode = checkMode
	state.MaxErrorCount = maxErrorCount
	state.FailEarly = extutil.ToBool(request.Config["failEarly"])
	state.Cursors = make(map[string]*ContainerLogCursor)
	return nil, nil
}

func (f LogErrorCountCheckAction) Start(_ context.Context, state *LogErrorCountCheckState) (*action_kit_api.StartResult, error) {
	// Only count what is logged after the step has started.
	now := time.Now()
	duration := state.End.Sub(state.Start)
	state.Start = now
	state.End = now.Add(duration)
	return nil, nil
}

func (f LogErrorCountCheckAction) Status(_ context.Context, state *LogErrorCountCheckState) (*action_kit_api.StatusResult, error) {
	now := time.Now()
	pods, err := f.GetPods(f.Client, state.Namespace, state.Target)
	if err != nil {
		return nil, err
	}

	errorMatcher, err := compileErrorMatcher(state.ErrorPattern)
	if err != nil {
		return nil, extension_kit.ToError("Invalid error pattern.", err)
	}
	var ignoreRegex *regexp.Regexp
	if state.IgnorePattern != "" {
		if ignoreRegex, err = regexp.Compile(state.IgnorePattern); err != nil {
			return nil, extension_kit.ToError("Invalid ignore pattern.", err)
		}
	}

	streamer := f.logStreamer
	if streamer == nil {
		streamer = f.Client
	}
	watcher := getOrCreateLogWatcher(state, streamer, errorMatcher, ignoreRegex)
	watcher.reconcile(pods)
	if f.waitForReads {
		watcher.wait()
	}
	state.Cursors = watcher.snapshot()
	messages := watcher.drainWarnings()

	completed := !now.Before(state.End)
	checkError := evaluateLogErrorCount(state)

	result := &action_kit_api.StatusResult{
		Completed: completed,
		Metrics:   new(buildLogCountMetrics(state, pods, now)),
	}
	if checkError != nil && (state.FailEarly || completed) {
		result.Completed = true
		result.Error = checkError
	}
	if result.Completed {
		stopLogWatcher(state.ExecutionId)
		messages = append(messages, action_kit_api.Message{
			Level:   extutil.Ptr(action_kit_api.Info),
			Message: summarizeLogCounts(state),
		})
	}
	if len(messages) > 0 {
		result.Messages = &messages
	}
	return result, nil
}

func restartCount(pod *corev1.Pod, containerName string) int32 {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			return status.RestartCount
		}
	}
	return 0
}

func isContainerRunning(pod *corev1.Pod, containerName string) bool {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			return status.State.Running != nil
		}
	}
	return false
}

func (f LogErrorCountCheckAction) Stop(_ context.Context, state *LogErrorCountCheckState) (*action_kit_api.StopResult, error) {
	stopLogWatcher(state.ExecutionId)
	return nil, nil
}

// countLogLines consumes a log stream produced with `timestamps=true`. Each line starts with an RFC3339Nano
// timestamp followed by a space. Lines that were already counted in a previous call are skipped.
//
// Lines are compared against the position of the previous call only (not against the previous line), so
// slightly out-of-order timestamps within a read (e.g. interleaved stdout/stderr) are still counted.
//
// Counts and position are accumulated locally and flushed to the cursor (holding mu) in batches: every
// logFlushLines lines, whenever the stream has no more data buffered (i.e. before blocking on a followed stream)
// and at the end. Counts and position are always flushed together, so a snapshot of the cursor is consistent.
func countLogLines(stream io.Reader, cursor *ContainerLogCursor, errorMatcher lineMatcher, ignoreRegex *regexp.Regexp, mu sync.Locker) error {
	reader := bufio.NewReaderSize(stream, 64*1024)
	mu.Lock()
	startTime, startSeen := cursor.LastTime, cursor.SeenAtLast
	mu.Unlock()
	maxTime, atMax, seenAtStart := startTime, 0, 0
	processedAny := false
	pendingLines, pendingErrors := 0, 0
	lineBuffer := make([]byte, 0, 1024)

	flush := func() {
		mu.Lock()
		defer mu.Unlock()
		cursor.Lines += pendingLines
		cursor.Errors += pendingErrors
		if maxTime.Equal(startTime) {
			cursor.SeenAtLast = max(startSeen, atMax)
		} else {
			cursor.LastTime = maxTime
			cursor.SeenAtLast = atMax
		}
		pendingLines, pendingErrors = 0, 0
	}
	defer flush()

	for {
		line, terminated, err := readLine(reader, lineBuffer[:0])
		lineBuffer = line
		// A final line without line terminator may have been cut (or still be written). Leave it for the next
		// read, which returns it completely, unless it is the only line - then count it to make progress.
		if len(line) > 0 && (terminated || !processedAny) {
			processedAny = true
			if ts, message, ok := splitTimestamp(line); ok && !ts.Before(startTime) {
				alreadyCounted := false
				if ts.Equal(startTime) {
					seenAtStart++
					alreadyCounted = seenAtStart <= startSeen
				}
				if !alreadyCounted {
					pendingLines++
					if errorMatcher.Match(message) && (ignoreRegex == nil || !ignoreRegex.Match(message)) {
						pendingErrors++
					}
				}
				if ts.After(maxTime) {
					maxTime, atMax = ts, 1
				} else if ts.Equal(maxTime) {
					atMax++
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if pendingLines >= logFlushLines || reader.Buffered() == 0 {
			flush()
		}
	}
}

// readLine appends the next line (without the line terminator, truncated to logMaxLineLength) to buf and returns
// it, together with whether the line was terminated by a newline. The returned slice is only valid until the
// next call when buf is reused.
func readLine(reader *bufio.Reader, buf []byte) ([]byte, bool, error) {
	line := buf
	for {
		chunk, err := reader.ReadSlice('\n')
		terminated := err == nil
		if terminated {
			chunk = bytes.TrimRight(chunk, "\r\n")
		}
		if remaining := logMaxLineLength - len(line); remaining > 0 {
			if len(chunk) > remaining {
				chunk = chunk[:remaining]
			}
			line = append(line, chunk...)
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return line, terminated, err
	}
}

func splitTimestamp(line []byte) (time.Time, []byte, bool) {
	idx := bytes.IndexByte(line, ' ')
	if idx <= 0 {
		return time.Time{}, nil, false
	}
	// time.Parse does not keep the string on success, so the bytes can be passed without copying them.
	ts, err := time.Parse(time.RFC3339Nano, unsafe.String(&line[0], idx))
	if err != nil {
		return time.Time{}, nil, false
	}
	return ts, line[idx+1:], true
}

type podLogCount struct {
	errors int
	lines  int
}

func countsByPod(state *LogErrorCountCheckState) map[string]*podLogCount {
	result := make(map[string]*podLogCount)
	for _, cursor := range state.Cursors {
		c, ok := result[cursor.Pod]
		if !ok {
			c = &podLogCount{}
			result[cursor.Pod] = c
		}
		c.errors += cursor.Errors
		c.lines += cursor.Lines
	}
	return result
}

func evaluateLogErrorCount(state *LogErrorCountCheckState) *action_kit_api.ActionKitError {
	switch state.CheckMode {
	case LogErrorCountCheckModeTotalExceeded:
		total := 0
		for _, c := range countsByPod(state) {
			total += c.errors
		}
		if total > state.MaxErrorCount {
			return &action_kit_api.ActionKitError{
				Title:  fmt.Sprintf("%s/%s logged %d error lines, max allowed is %d.", state.Namespace, state.Target, total, state.MaxErrorCount),
				Status: extutil.Ptr(action_kit_api.Failed),
			}
		}
	case LogErrorCountCheckModePerPodExceeded:
		counts := countsByPod(state)
		var exceeded []string
		for pod, c := range counts {
			if c.errors > state.MaxErrorCount {
				exceeded = append(exceeded, fmt.Sprintf("%s (%d)", pod, c.errors))
			}
		}
		if len(exceeded) > 0 {
			sort.Strings(exceeded)
			return &action_kit_api.ActionKitError{
				Title:  fmt.Sprintf("%s/%s has pods with more than %d error log lines: %s.", state.Namespace, state.Target, state.MaxErrorCount, strings.Join(exceeded, ", ")),
				Status: extutil.Ptr(action_kit_api.Failed),
			}
		}
	}
	return nil
}

// buildLogCountMetrics emits the cumulative counts of all pods currently belonging to the target.
func buildLogCountMetrics(state *LogErrorCountCheckState, pods []*corev1.Pod, now time.Time) []action_kit_api.Metric {
	counts := countsByPod(state)
	metrics := make([]action_kit_api.Metric, 0, len(pods)*2)
	for _, pod := range pods {
		c, ok := counts[pod.Name]
		if !ok {
			continue
		}
		labels := map[string]string{
			"k8s.cluster-name":   extconfig.Config.ClusterName,
			"k8s.namespace":      state.Namespace,
			state.MetricLabelKey: state.Target,
			"k8s.pod.name":       pod.Name,
		}
		metrics = append(metrics,
			action_kit_api.Metric{Name: new(LogErrorCountMetricName), Metric: labels, Timestamp: now, Value: float64(c.errors)},
			action_kit_api.Metric{Name: new(LogLineCountMetricName), Metric: labels, Timestamp: now, Value: float64(c.lines)},
		)
	}
	return metrics
}

func summarizeLogCounts(state *LogErrorCountCheckState) string {
	counts := countsByPod(state)
	pods := make([]string, 0, len(counts))
	totalErrors, totalLines := 0, 0
	for pod, c := range counts {
		pods = append(pods, pod)
		totalErrors += c.errors
		totalLines += c.lines
	}
	sort.Strings(pods)
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s/%s logged %d error lines out of %d lines", state.Namespace, state.Target, totalErrors, totalLines)
	for i, pod := range pods {
		if i == 0 {
			sb.WriteString(": ")
		} else {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%s %d/%d", pod, counts[pod].errors, counts[pod].lines)
	}
	return sb.String()
}

// NewWorkloadLogErrorCountCheckAction creates the error log count check for a workload kind. The pods are
// resolved via their owner references (deployments via their ReplicaSets), like the discoveries do.
func NewWorkloadLogErrorCountCheckAction(k8s *client.Client, actionId, targetType, kindLabel, attributeKey string, viaReplicaSets bool, getWorkload func(k8s *client.Client, namespace, name string) metav1.Object) action_kit_sdk.Action[LogErrorCountCheckState] {
	kind := strings.ToLower(kindLabel)
	return &LogErrorCountCheckAction{
		Client:          k8s,
		ActionId:        actionId,
		TargetType:      targetType,
		TargetTypeLabel: kindLabel,
		SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       kind,
				Description: new(fmt.Sprintf("Find %s by cluster, namespace and %s", kind, kind)),
				Query:       fmt.Sprintf("k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND %s=\"\"", attributeKey),
			},
		}),
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes[attributeKey][0]
		},
		GetPods: func(k8s *client.Client, namespace string, target string) ([]*corev1.Pod, error) {
			workload := getWorkload(k8s, namespace, target)
			if workload == nil || reflect.ValueOf(workload).IsNil() {
				return nil, extension_kit.ToError(fmt.Sprintf("%s %s not found.", kindLabel, target), nil)
			}
			return k8s.PodsOfWorkload(workload.GetUID(), namespace, viaReplicaSets), nil
		},
		MetricLabelKey: attributeKey,
	}
}
