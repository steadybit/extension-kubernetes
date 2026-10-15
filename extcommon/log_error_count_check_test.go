// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type logLine struct {
	ts      time.Time
	message string
}

// fakeLogStreamer emulates the pods/log endpoint: it honours sinceTime with second precision only
// (like the real API) and always prefixes lines with their RFC3339Nano timestamp.
type fakeLogStreamer struct {
	mu       sync.Mutex
	logs     map[string][]logLine
	previous map[string][]logLine
	failures map[string]error
	calls    int
}

func newFakeLogStreamer() *fakeLogStreamer {
	return &fakeLogStreamer{logs: map[string][]logLine{}, previous: map[string][]logLine{}, failures: map[string]error{}}
}

func (s *fakeLogStreamer) add(pod, container string, ts time.Time, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := pod + "/" + container
	s.logs[key] = append(s.logs[key], logLine{ts: ts, message: message})
}

func (s *fakeLogStreamer) PodLogs(_ context.Context, _ string, podName string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	key := podName + "/" + opts.Container
	if err := s.failures[key]; err != nil {
		return nil, err
	}
	var since time.Time
	if opts.SinceTime != nil {
		since = opts.SinceTime.Time.Truncate(time.Second)
	}
	lines := s.logs[key]
	if opts.Previous {
		lines = s.previous[key]
	}
	var sb strings.Builder
	for _, l := range lines {
		if l.ts.Before(since) {
			continue
		}
		sb.WriteString(l.ts.UTC().Format(time.RFC3339Nano))
		sb.WriteString(" ")
		sb.WriteString(l.message)
		sb.WriteString("\n")
	}
	out := sb.String()
	if opts.LimitBytes != nil && int64(len(out)) > *opts.LimitBytes {
		out = out[:*opts.LimitBytes]
	}
	return io.NopCloser(strings.NewReader(out)), nil
}

// restart simulates a container restart: the current logs become the previous instance's logs.
func (s *fakeLogStreamer) restart(pod *corev1.Pod, container string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := pod.Name + "/" + container
	s.previous[key] = s.logs[key]
	s.logs[key] = nil
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == container {
			pod.Status.ContainerStatuses[i].RestartCount++
		}
	}
}

func testPod(name string, containers ...string) *corev1.Pod {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "shop"}}
	for _, c := range containers {
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: c})
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
			Name:  c,
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
		})
	}
	return pod
}

func newTestLogErrorCountCheck(streamer PodLogStreamer, pods *[]*corev1.Pod) LogErrorCountCheckAction {
	return LogErrorCountCheckAction{
		ActionId:        "test",
		TargetType:      "test-type",
		TargetTypeLabel: "Deployment",
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes["k8s.deployment"][0]
		},
		GetPods: func(_ *client.Client, _ string, _ string) ([]*corev1.Pod, error) {
			return *pods, nil
		},
		MetricLabelKey: "k8s.deployment",
		logStreamer:    streamer,
		waitForReads:   true,
	}
}

func setLogStreamLimits(t *testing.T, maxPerStep, maxConcurrent int) {
	previousPerStep, previousConcurrent := extconfig.Config.LogErrorCountMaxContainersPerStep, extconfig.Config.LogErrorCountMaxConcurrentStreams
	extconfig.Config.LogErrorCountMaxContainersPerStep = maxPerStep
	extconfig.Config.LogErrorCountMaxConcurrentStreams = maxConcurrent
	t.Cleanup(func() {
		extconfig.Config.LogErrorCountMaxContainersPerStep = previousPerStep
		extconfig.Config.LogErrorCountMaxConcurrentStreams = previousConcurrent
	})
}

func TestMain(m *testing.M) {
	extconfig.Config.LogErrorCountMaxContainersPerStep = 20
	extconfig.Config.LogErrorCountMaxConcurrentStreams = 100
	m.Run()
}

func prepareRequest(config map[string]any) action_kit_api.PrepareActionRequestBody {
	return action_kit_api.PrepareActionRequestBody{
		ExecutionId: uuid.New(),
		Config:      config,
		Target: new(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace":  {"shop"},
				"k8s.deployment": {"checkout"},
			},
		}),
	}
}

func metricValue(metrics *[]action_kit_api.Metric, name, pod string) (float64, bool) {
	if metrics == nil {
		return 0, false
	}
	for _, m := range *metrics {
		if *m.Name == name && m.Metric["k8s.pod.name"] == pod {
			return m.Value, true
		}
	}
	return 0, false
}

func TestLogErrorCountCheck_PrepareDefaults(t *testing.T) {
	pods := []*corev1.Pod{testPod("checkout-1", "app")}
	action := newTestLogErrorCountCheck(newFakeLogStreamer(), &pods)
	state := action.NewEmptyState()

	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 10_000}))

	require.NoError(t, err)
	assert.Equal(t, LogErrorCountCheckModeNone, state.CheckMode)
	assert.Equal(t, DefaultLogErrorPattern, state.ErrorPattern)
	assert.Equal(t, "shop", state.Namespace)
	assert.Equal(t, "checkout", state.Target)
	assert.Equal(t, "k8s.deployment", state.MetricLabelKey)
	assert.False(t, state.FailEarly)
	assert.Equal(t, 10*time.Second, state.End.Sub(state.Start))
}

func TestLogErrorCountCheck_PrepareRejectsInvalidConfig(t *testing.T) {
	pods := []*corev1.Pod{testPod("checkout-1", "app")}
	action := newTestLogErrorCountCheck(newFakeLogStreamer(), &pods)

	for name, config := range map[string]map[string]any{
		"invalid error pattern":  {"duration": 1000, "errorPattern": "("},
		"invalid ignore pattern": {"duration": 1000, "ignorePattern": "["},
		"unknown check mode":     {"duration": 1000, "checkMode": "whatever"},
		"negative max":           {"duration": 1000, "checkMode": string(LogErrorCountCheckModeTotalExceeded), "maxErrorCount": -1},
	} {
		t.Run(name, func(t *testing.T) {
			state := action.NewEmptyState()
			_, err := action.Prepare(context.Background(), &state, prepareRequest(config))
			require.Error(t, err)
		})
	}
}

func TestLogErrorCountCheck_CountsOnlyNewLinesAcrossStatusCalls(t *testing.T) {
	streamer := newFakeLogStreamer()
	pods := []*corev1.Pod{testPod("checkout-1", "app", "sidecar"), testPod("checkout-2", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 60_000}))
	require.NoError(t, err)
	_, err = action.Start(context.Background(), &state)
	require.NoError(t, err)

	// logged before the step started -> ignored
	streamer.add("checkout-1", "app", state.Start.Add(-2*time.Second), "ERROR before start")

	base := state.Start.Add(10 * time.Millisecond)
	streamer.add("checkout-1", "app", base, "ERROR failed to connect to db")
	streamer.add("checkout-1", "app", base.Add(time.Millisecond), "INFO request served")
	streamer.add("checkout-1", "sidecar", base, `{"level":"error","msg":"upstream reset"}`)
	streamer.add("checkout-2", "app", base, "level=info msg=ok")

	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	assert.False(t, result.Completed)
	assert.Nil(t, result.Error)

	v, ok := metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	require.True(t, ok)
	assert.Equal(t, float64(2), v, "app + sidecar errors are aggregated per pod")
	v, _ = metricValue(result.Metrics, LogLineCountMetricName, "checkout-1")
	assert.Equal(t, float64(3), v)
	v, ok = metricValue(result.Metrics, LogErrorCountMetricName, "checkout-2")
	require.True(t, ok)
	assert.Equal(t, float64(0), v)

	// new lines within the same second as the last seen line must be counted exactly once
	streamer.add("checkout-1", "app", base.Add(2*time.Millisecond), "panic: runtime error")
	streamer.add("checkout-2", "app", base.Add(3*time.Second), "java.lang.NullPointerException: boom")

	result, err = action.Status(context.Background(), &state)
	require.NoError(t, err)
	v, _ = metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	assert.Equal(t, float64(3), v)
	v, _ = metricValue(result.Metrics, LogLineCountMetricName, "checkout-1")
	assert.Equal(t, float64(4), v)
	v, _ = metricValue(result.Metrics, LogErrorCountMetricName, "checkout-2")
	assert.Equal(t, float64(1), v)

	// nothing new -> counts stay
	result, err = action.Status(context.Background(), &state)
	require.NoError(t, err)
	v, _ = metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	assert.Equal(t, float64(3), v)
}

func TestLogErrorCountCheck_PicksUpNewPodsAndIgnoresGonePodsInMetrics(t *testing.T) {
	streamer := newFakeLogStreamer()
	pods := []*corev1.Pod{testPod("checkout-1", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{
		"duration":      60_000,
		"checkMode":     string(LogErrorCountCheckModeTotalExceeded),
		"maxErrorCount": 1,
	}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)

	streamer.add("checkout-1", "app", state.Start.Add(time.Millisecond), "ERROR one")
	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	assert.Nil(t, result.Error)

	// rollout: old pod is replaced by a new one which logs an error
	pods = []*corev1.Pod{testPod("checkout-2", "app")}
	streamer.add("checkout-2", "app", state.Start.Add(time.Second), "ERROR two")
	result, err = action.Status(context.Background(), &state)
	require.NoError(t, err)

	_, ok := metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	assert.False(t, ok, "gone pods are not reported anymore")
	v, ok := metricValue(result.Metrics, LogErrorCountMetricName, "checkout-2")
	require.True(t, ok)
	assert.Equal(t, float64(1), v)

	// errors of the gone pod still count towards the total; not fail early -> not completed yet
	assert.False(t, result.Completed)
	assert.Nil(t, result.Error)

	state.End = time.Now()
	result, err = action.Status(context.Background(), &state)
	require.NoError(t, err)
	assert.True(t, result.Completed)
	require.NotNil(t, result.Error)
	assert.Equal(t, action_kit_api.Failed, *result.Error.Status)
	assert.Contains(t, result.Error.Title, "logged 2 error lines, max allowed is 1")
	require.NotNil(t, result.Messages)
	assert.Contains(t, (*result.Messages)[len(*result.Messages)-1].Message, "checkout-1 1/1, checkout-2 1/1")
}

func TestLogErrorCountCheck_FailEarlyPerPod(t *testing.T) {
	streamer := newFakeLogStreamer()
	pods := []*corev1.Pod{testPod("checkout-1", "app"), testPod("checkout-2", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{
		"duration":      60_000,
		"checkMode":     string(LogErrorCountCheckModePerPodExceeded),
		"maxErrorCount": 1,
		"failEarly":     true,
	}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)

	streamer.add("checkout-1", "app", state.Start.Add(time.Millisecond), "ERROR one")
	streamer.add("checkout-2", "app", state.Start.Add(time.Millisecond), "ERROR one")
	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	assert.False(t, result.Completed, "1 error per pod is within the limit even though the total is 2")

	streamer.add("checkout-2", "app", state.Start.Add(2*time.Millisecond), "ERROR two")
	result, err = action.Status(context.Background(), &state)
	require.NoError(t, err)
	assert.True(t, result.Completed)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Title, "checkout-2 (2)")
	assert.NotContains(t, result.Error.Title, "checkout-1")
}

func TestLogErrorCountCheck_ModeNoneNeverFails(t *testing.T) {
	streamer := newFakeLogStreamer()
	pods := []*corev1.Pod{testPod("checkout-1", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 0, "failEarly": true}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)
	for i := 0; i < 10; i++ {
		streamer.add("checkout-1", "app", state.Start.Add(time.Duration(i)*time.Millisecond), fmt.Sprintf("ERROR %d", i))
	}

	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	assert.True(t, result.Completed)
	assert.Nil(t, result.Error)
	v, _ := metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	assert.Equal(t, float64(10), v)
}

func TestLogErrorCountCheck_ContainerFilterAndIgnorePattern(t *testing.T) {
	streamer := newFakeLogStreamer()
	pods := []*corev1.Pod{testPod("checkout-1", "app", "istio-proxy")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{
		"duration":      60_000,
		"containerName": "app",
		"ignorePattern": "health",
	}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)

	streamer.add("checkout-1", "app", state.Start.Add(time.Millisecond), "ERROR healthcheck flaky")
	streamer.add("checkout-1", "app", state.Start.Add(2*time.Millisecond), "ERROR real problem")
	streamer.add("checkout-1", "istio-proxy", state.Start.Add(time.Millisecond), "ERROR upstream")

	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	v, _ := metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	assert.Equal(t, float64(1), v)
	v, _ = metricValue(result.Metrics, LogLineCountMetricName, "checkout-1")
	assert.Equal(t, float64(2), v, "sidecar lines are not read")
}

func TestLogErrorCountCheck_ReportsLogReadFailuresAsWarnings(t *testing.T) {
	streamer := newFakeLogStreamer()
	streamer.failures["checkout-1/app"] = errors.New("forbidden")
	pods := []*corev1.Pod{testPod("checkout-1", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 60_000}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)

	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	assert.Nil(t, result.Error)
	require.NotNil(t, result.Messages)
	assert.Equal(t, action_kit_api.Warn, *(*result.Messages)[0].Level)
	assert.Contains(t, (*result.Messages)[0].Message, "forbidden")
}

func TestCountLogLines_TruncatesLongLinesAndSkipsUnparsable(t *testing.T) {
	ts := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	long := strings.Repeat("x", logMaxLineLength*2) + " ERROR"
	input := strings.Join([]string{
		ts.Format(time.RFC3339Nano) + " " + long,
		"not-a-timestamp ERROR",
		ts.Add(time.Millisecond).Format(time.RFC3339Nano) + " ERROR short",
		"",
	}, "\n")
	cursor := &ContainerLogCursor{LastTime: ts}

	err := countLogLines(strings.NewReader(input), cursor, regexp.MustCompile(DefaultLogErrorPattern), nil, &sync.Mutex{})

	require.NoError(t, err)
	assert.Equal(t, 2, cursor.Lines)
	assert.Equal(t, 1, cursor.Errors, "the ERROR beyond the max line length is cut off")
	assert.Equal(t, ts.Add(time.Millisecond), cursor.LastTime)
	assert.Equal(t, 1, cursor.SeenAtLast)
}

func TestCountLogLines_DoesNotCountTruncatedLastLine(t *testing.T) {
	ts := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	first := ts.Format(time.RFC3339Nano) + " INFO ok\n"
	second := ts.Add(time.Millisecond).Format(time.RFC3339Nano) + " something went wrong: ERROR\n"
	cursor := &ContainerLogCursor{LastTime: ts}
	errorRegex := regexp.MustCompile(DefaultLogErrorPattern)

	// the stream is cut (LimitBytes) in the middle of the second line, before the error marker
	err := countLogLines(strings.NewReader(first+second[:len(second)-10]), cursor, errorRegex, nil, &sync.Mutex{})
	require.NoError(t, err)
	assert.Equal(t, 1, cursor.Lines)
	assert.Equal(t, 0, cursor.Errors)

	// the next call (sinceTime truncated to the second) returns both lines completely
	err = countLogLines(strings.NewReader(first+second), cursor, errorRegex, nil, &sync.Mutex{})
	require.NoError(t, err)
	assert.Equal(t, 2, cursor.Lines)
	assert.Equal(t, 1, cursor.Errors)
}

func TestCountLogLines_CountsSingleUnterminatedLineToMakeProgress(t *testing.T) {
	ts := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	cursor := &ContainerLogCursor{LastTime: ts}

	err := countLogLines(strings.NewReader(ts.Format(time.RFC3339Nano)+" ERROR no newline"), cursor, regexp.MustCompile(DefaultLogErrorPattern), nil, &sync.Mutex{})

	require.NoError(t, err)
	assert.Equal(t, 1, cursor.Lines)
	assert.Equal(t, 1, cursor.Errors)
}

func TestLogErrorCountCheck_CountsLogsOfCrashedContainer(t *testing.T) {
	streamer := newFakeLogStreamer()
	pod := testPod("checkout-1", "app")
	pod.Status.ContainerStatuses[0].RestartCount = 3 // restarts before the step are ignored
	pods := []*corev1.Pod{pod}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 60_000}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)

	streamer.add("checkout-1", "app", state.Start.Add(time.Millisecond), "ERROR first")
	_, err = action.Status(context.Background(), &state)
	require.NoError(t, err)

	// the container logs a fatal error and crashes before the next status call
	streamer.add("checkout-1", "app", state.Start.Add(2*time.Second), "FATAL out of memory")
	streamer.restart(pod, "app")
	streamer.add("checkout-1", "app", state.Start.Add(3*time.Second), "INFO started again")

	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	v, _ := metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	assert.Equal(t, float64(2), v)
	v, _ = metricValue(result.Metrics, LogLineCountMetricName, "checkout-1")
	assert.Equal(t, float64(3), v)
}

func TestLogErrorCountCheck_ReadsLargeBacklog(t *testing.T) {
	streamer := newFakeLogStreamer()
	pods := []*corev1.Pod{testPod("checkout-1", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 60_000}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)

	// ~10 MiB, spread over several seconds
	payload := strings.Repeat("x", 1000)
	lines := 10 * 1024 * 1024 / 1040
	for i := 0; i < lines; i++ {
		streamer.add("checkout-1", "app", state.Start.Add(time.Duration(i)*time.Millisecond), "ERROR "+payload)
	}

	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	v, _ := metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	assert.Equal(t, float64(lines), v, "all lines are read")
}

func TestCountLogLines_CountsOutOfOrderLinesWithinARead(t *testing.T) {
	ts := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	input := strings.Join([]string{
		ts.Add(5*time.Millisecond).Format(time.RFC3339Nano) + " INFO stdout",
		ts.Add(4*time.Millisecond).Format(time.RFC3339Nano) + " ERROR stderr, stamped slightly earlier",
		"",
	}, "\n")
	cursor := &ContainerLogCursor{LastTime: ts}
	errorRegex := regexp.MustCompile(DefaultLogErrorPattern)

	require.NoError(t, countLogLines(strings.NewReader(input), cursor, errorRegex, nil, &sync.Mutex{}))
	assert.Equal(t, 2, cursor.Lines)
	assert.Equal(t, 1, cursor.Errors)
	assert.Equal(t, ts.Add(5*time.Millisecond), cursor.LastTime)

	// reading the same second again does not count anything twice
	require.NoError(t, countLogLines(strings.NewReader(input), cursor, errorRegex, nil, &sync.Mutex{}))
	assert.Equal(t, 2, cursor.Lines)
	assert.Equal(t, 1, cursor.Errors)
}

func TestLogErrorCountCheck_PrepareRejectsTooManyContainers(t *testing.T) {
	setLogStreamLimits(t, 2, 100)
	pods := []*corev1.Pod{testPod("checkout-1", "app", "sidecar"), testPod("checkout-2", "app")}
	action := newTestLogErrorCountCheck(newFakeLogStreamer(), &pods)

	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 1000}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 2 containers")

	// restricting to one container per pod fits into the limit
	state = action.NewEmptyState()
	_, err = action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 1000, "containerName": "app"}))
	require.NoError(t, err)
}

func TestLogErrorCountCheck_PrepareRejectsWhenGlobalLimitIsReached(t *testing.T) {
	setLogStreamLimits(t, 20, 2)
	openLogStreams.Lock()
	openLogStreams.count = 1
	openLogStreams.Unlock()
	t.Cleanup(func() {
		openLogStreams.Lock()
		openLogStreams.count = 0
		openLogStreams.Unlock()
	})
	pods := []*corev1.Pod{testPod("checkout-1", "app"), testPod("checkout-2", "app")}
	action := newTestLogErrorCountCheck(newFakeLogStreamer(), &pods)

	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 1000}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 already in use")
}

func TestLogErrorCountCheck_SkipsContainersBeyondTheLimitDuringTheStep(t *testing.T) {
	setLogStreamLimits(t, 1, 100)
	streamer := &blockingLogStreamer{fakeLogStreamer: newFakeLogStreamer()}
	pods := []*corev1.Pod{testPod("checkout-1", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	action.waitForReads = false
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 60_000}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)
	t.Cleanup(func() { stopLogWatcher(state.ExecutionId) })

	_, err = action.Status(context.Background(), &state)
	require.NoError(t, err)

	// a rollout adds a second pod while the first one is still followed
	pods = append(pods, testPod("checkout-2", "app"))
	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	require.NotNil(t, result.Messages)
	require.Len(t, *result.Messages, 1)
	assert.Contains(t, (*result.Messages)[0].Message, "Not counting the logs of pod checkout-2")

	// the warning is reported once only
	result, err = action.Status(context.Background(), &state)
	require.NoError(t, err)
	assert.Nil(t, result.Messages)
}

func TestLogErrorCountCheck_ClosesStreamsOnStop(t *testing.T) {
	streamer := &blockingLogStreamer{fakeLogStreamer: newFakeLogStreamer()}
	pods := []*corev1.Pod{testPod("checkout-1", "app"), testPod("checkout-2", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	action.waitForReads = false
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 60_000}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)

	_, err = action.Status(context.Background(), &state)
	require.NoError(t, err)
	before := openLogStreamCount()
	require.GreaterOrEqual(t, before, 2)

	_, err = action.Stop(context.Background(), &state)
	require.NoError(t, err)
	// <= as streams of previous tests may still be closing in the background
	assert.Eventually(t, func() bool { return openLogStreamCount() <= before-2 }, time.Second, 10*time.Millisecond)
	logWatchers.Lock()
	assert.NotContains(t, logWatchers.m, state.ExecutionId)
	logWatchers.Unlock()
}

func TestLogErrorCountCheck_ResumesFromStateAfterExtensionRestart(t *testing.T) {
	streamer := newFakeLogStreamer()
	pods := []*corev1.Pod{testPod("checkout-1", "app")}
	action := newTestLogErrorCountCheck(streamer, &pods)
	state := action.NewEmptyState()
	_, err := action.Prepare(context.Background(), &state, prepareRequest(map[string]any{"duration": 60_000}))
	require.NoError(t, err)
	_, _ = action.Start(context.Background(), &state)

	streamer.add("checkout-1", "app", state.Start.Add(time.Millisecond), "ERROR one")
	_, err = action.Status(context.Background(), &state)
	require.NoError(t, err)

	// the extension restarts: the in-memory watcher is gone, the state is sent again by the platform
	stopLogWatcher(state.ExecutionId)
	streamer.add("checkout-1", "app", state.Start.Add(2*time.Millisecond), "ERROR two")

	result, err := action.Status(context.Background(), &state)
	require.NoError(t, err)
	v, _ := metricValue(result.Metrics, LogErrorCountMetricName, "checkout-1")
	assert.Equal(t, float64(2), v, "counts are resumed from the state without counting lines twice")
}

// blockingLogStreamer returns streams that stay open (like follow=true) until their context is cancelled.
type blockingLogStreamer struct {
	*fakeLogStreamer
}

func (s *blockingLogStreamer) PodLogs(ctx context.Context, _ string, _ string, _ *corev1.PodLogOptions) (io.ReadCloser, error) {
	reader, writer := io.Pipe()
	go func() {
		<-ctx.Done()
		_ = writer.Close()
	}()
	return reader, nil
}

func TestDefaultLogErrorPattern(t *testing.T) {
	pattern := regexp.MustCompile(DefaultLogErrorPattern)
	for _, line := range []string{
		"ERROR something failed",
		"2026-10-15 [error] boom",
		`{"level":"error","msg":"x"}`,
		"level=ERROR msg=x",
		"FATAL cannot start",
		"panic: runtime error: index out of range",
		"java.lang.NullPointerException: x",
		"Exception in thread \"main\"",
		"CRITICAL disk full",
	} {
		assert.True(t, pattern.MatchString(line), line)
	}
	for _, line := range []string{
		"INFO request served",
		"errors=0",
		"error_count=0",
		"stderr redirected",
		`{"level":"info","msg":"x"}`,
	} {
		assert.False(t, pattern.MatchString(line), line)
	}
}
