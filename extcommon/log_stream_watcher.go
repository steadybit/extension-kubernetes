// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Stream slots per step execution, limited by extconfig.Config.LogErrorCountMaxConcurrentStreams in total to bound
// the load on the API server and the kubelets. Prepare reserves the slots for the containers a step has at that
// time, so steps started later can't take them away; streams beyond the reservation (e.g. pods added by a rollout)
// use free slots only. A step claims max(reserved, open) slots.
var logStreamSlots = struct {
	sync.Mutex
	steps map[uuid.UUID]*logStreamSlotUsage
}{steps: map[uuid.UUID]*logStreamSlotUsage{}}

type logStreamSlotUsage struct {
	reserved int
	open     int
	// expires releases the reservation of a step that is never started or stopped
	expires time.Time
}

func (u *logStreamSlotUsage) claimed() int {
	return max(u.reserved, u.open)
}

// claimedLogStreamSlots must be called holding logStreamSlots.
func claimedLogStreamSlots(now time.Time) int {
	total := 0
	for executionId, u := range logStreamSlots.steps {
		if u.open == 0 && now.After(u.expires) {
			delete(logStreamSlots.steps, executionId)
			continue
		}
		total += u.claimed()
	}
	return total
}

// reserveLogStreamSlots reserves the slots for a step following the given number of containers, or returns an error
// if the step exceeds the per-step limit or the slots are not available.
func reserveLogStreamSlots(executionId uuid.UUID, containers int, expires time.Time) error {
	maxPerStep := extconfig.Config.LogErrorCountMaxContainersPerStep
	if containers > maxPerStep {
		return extension_kit.ToError(fmt.Sprintf("The target has %d containers, but the logs of at most %d containers can be followed per step. Narrow down the target selection or the container name, or raise the limit via STEADYBIT_EXTENSION_LOG_ERROR_COUNT_MAX_CONTAINERS_PER_STEP.", containers, maxPerStep), nil)
	}
	logStreamSlots.Lock()
	defer logStreamSlots.Unlock()
	maxStreams := extconfig.Config.LogErrorCountMaxConcurrentStreams
	used := claimedLogStreamSlots(time.Now())
	u, ok := logStreamSlots.steps[executionId]
	if ok {
		used -= u.claimed()
	} else {
		u = &logStreamSlotUsage{}
	}
	if used+containers > maxStreams {
		return extension_kit.ToError(fmt.Sprintf("Following %d more containers would exceed the limit of %d concurrently followed containers (%d already in use by other steps). Raise the limit via STEADYBIT_EXTENSION_LOG_ERROR_COUNT_MAX_CONCURRENT_STREAMS.", containers, maxStreams, used), nil)
	}
	u.reserved = containers
	u.expires = expires
	logStreamSlots.steps[executionId] = u
	return nil
}

func acquireLogStreamSlot(executionId uuid.UUID) bool {
	logStreamSlots.Lock()
	defer logStreamSlots.Unlock()
	used := claimedLogStreamSlots(time.Now())
	u, ok := logStreamSlots.steps[executionId]
	if !ok {
		u = &logStreamSlotUsage{}
		logStreamSlots.steps[executionId] = u
	}
	if used-u.claimed()+max(u.reserved, u.open+1) > extconfig.Config.LogErrorCountMaxConcurrentStreams {
		return false
	}
	u.open++
	return true
}

func releaseLogStreamSlot(executionId uuid.UUID) {
	logStreamSlots.Lock()
	defer logStreamSlots.Unlock()
	if u, ok := logStreamSlots.steps[executionId]; ok {
		u.open--
		if u.open <= 0 && u.reserved == 0 {
			delete(logStreamSlots.steps, executionId)
		}
	}
}

// releaseLogStreamReservation gives up the reservation of a step; slots of streams still closing are released by them.
func releaseLogStreamReservation(executionId uuid.UUID) {
	logStreamSlots.Lock()
	defer logStreamSlots.Unlock()
	if u, ok := logStreamSlots.steps[executionId]; ok {
		u.reserved = 0
		if u.open <= 0 {
			delete(logStreamSlots.steps, executionId)
		}
	}
}

func openLogStreamCount() int {
	logStreamSlots.Lock()
	defer logStreamSlots.Unlock()
	open := 0
	for _, u := range logStreamSlots.steps {
		open += u.open
	}
	return open
}

// followedContainers returns the containers whose logs are counted: the regular containers and native sidecars
// (init containers with restartPolicy Always), optionally restricted to a single container name.
func followedContainers(pod *corev1.Pod, containerName string) []string {
	var names []string
	for _, c := range pod.Spec.InitContainers {
		if c.RestartPolicy != nil && *c.RestartPolicy == corev1.ContainerRestartPolicyAlways && (containerName == "" || c.Name == containerName) {
			names = append(names, c.Name)
		}
	}
	for _, c := range pod.Spec.Containers {
		if containerName == "" || c.Name == containerName {
			names = append(names, c.Name)
		}
	}
	return names
}

// countContainers counts the containers to follow of the pods that haven't terminated.
func countContainers(pods []*corev1.Pod, containerName string) int {
	count := 0
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		count += len(followedContainers(pod, containerName))
	}
	return count
}

func containerStatus(pod *corev1.Pod, containerName string) *corev1.ContainerStatus {
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == containerName {
			return &pod.Status.ContainerStatuses[i]
		}
	}
	for i := range pod.Status.InitContainerStatuses {
		if pod.Status.InitContainerStatuses[i].Name == containerName {
			return &pod.Status.InitContainerStatuses[i]
		}
	}
	return nil
}

func restartCount(pod *corev1.Pod, containerName string) int32 {
	if status := containerStatus(pod, containerName); status != nil {
		return status.RestartCount
	}
	return 0
}

func isContainerRunning(pod *corev1.Pod, containerName string) bool {
	status := containerStatus(pod, containerName)
	return status != nil && status.State.Running != nil
}

const (
	logStreamRetryMinBackoff = 2 * time.Second
	logStreamRetryMaxBackoff = 30 * time.Second
)

type logStreamRetry struct {
	failures  int
	notBefore time.Time
}

// logWatcher follows the logs of the containers of a single step execution (one `follow=true` stream per container).
type logWatcher struct {
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	// streamsCtx is cancelled to close the streams without ending the watcher (log volume budget exceeded).
	streamsCtx    context.Context
	cancelStreams context.CancelFunc
	// bytesRead counts the bytes read since the last log volume sample.
	bytesRead atomic.Int64
	// volumeWindow holds the bytes read per second over the last logVolumeWindowSeconds (guarded by logWatchers).
	volumeWindow   [logVolumeWindowSeconds]int64
	volumeIndex    int
	budgetExceeded bool
	wg             sync.WaitGroup
	streamer       PodLogStreamer
	executionId    uuid.UUID
	start          time.Time
	end            time.Time
	namespace      string
	target         string
	containerName  string
	errorMatcher   lineMatcher
	ignoreRegex    *regexp.Regexp
	cursors        map[string]*ContainerLogCursor
	active         map[string]bool
	retries        map[string]*logStreamRetry
	// incomplete is set when containers couldn't be followed (limits, log volume budget), so a passing result
	// can't be trusted.
	incomplete bool
	// warnings per container key; only reported when they change
	warnings         map[string]string
	reportedWarnings map[string]string
}

var logWatchers = struct {
	sync.Mutex
	m map[uuid.UUID]*logWatcher
}{m: map[uuid.UUID]*logWatcher{}}

// getOrCreateLogWatcher returns the watcher of the execution. If there is none (first status call, or the extension
// was restarted during the step), it is created from the cursors snapshot in the state.
func getOrCreateLogWatcher(state *LogErrorCountCheckState, streamer PodLogStreamer) (*logWatcher, error) {
	logWatchers.Lock()
	defer logWatchers.Unlock()
	if w, ok := logWatchers.m[state.ExecutionId]; ok {
		return w, nil
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
	ctx, cancel := context.WithDeadline(context.Background(), state.End.Add(logStreamGracePeriod))
	streamsCtx, cancelStreams := context.WithCancel(ctx)
	w := &logWatcher{
		ctx:              ctx,
		cancel:           cancel,
		streamsCtx:       streamsCtx,
		cancelStreams:    cancelStreams,
		budgetExceeded:   state.LogVolumeBudgetExceeded,
		incomplete:       state.Incomplete,
		streamer:         streamer,
		executionId:      state.ExecutionId,
		start:            state.Start,
		end:              state.End,
		target:           state.Target,
		namespace:        state.Namespace,
		containerName:    state.ContainerName,
		errorMatcher:     errorMatcher,
		ignoreRegex:      ignoreRegex,
		cursors:          make(map[string]*ContainerLogCursor),
		active:           make(map[string]bool),
		retries:          make(map[string]*logStreamRetry),
		warnings:         make(map[string]string),
		reportedWarnings: make(map[string]string),
	}
	for key, cursor := range state.Cursors {
		c := *cursor
		w.cursors[key] = &c
	}
	if w.budgetExceeded {
		cancelStreams()
	}
	logWatchers.m[state.ExecutionId] = w
	startLogVolumeMonitor()
	// Close the streams when the deadline is reached, even if stop is never called.
	context.AfterFunc(ctx, func() {
		removeLogWatcher(state.ExecutionId, w)
		releaseLogStreamReservation(state.ExecutionId)
	})
	return w, nil
}

func removeLogWatcher(executionId uuid.UUID, w *logWatcher) {
	logWatchers.Lock()
	defer logWatchers.Unlock()
	if logWatchers.m[executionId] == w {
		delete(logWatchers.m, executionId)
	}
}

func stopLogWatcher(executionId uuid.UUID) {
	logWatchers.Lock()
	w, ok := logWatchers.m[executionId]
	delete(logWatchers.m, executionId)
	logWatchers.Unlock()
	if ok {
		w.cancel()
	}
	releaseLogStreamReservation(executionId)
}

// reconcile opens a log stream for every running container of the given pods that isn't followed yet.
func (w *logWatcher) reconcile(pods []*corev1.Pod) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx.Err() != nil || w.budgetExceeded {
		return
	}
	now := time.Now()
	maxPerStep := extconfig.Config.LogErrorCountMaxContainersPerStep
	for _, pod := range pods {
		for _, containerName := range followedContainers(pod, w.containerName) {
			key := pod.Namespace + "/" + pod.Name + "/" + containerName
			// Only running containers are followed; a stream of a crashed container ends and is reopened once the
			// container runs again.
			if w.active[key] || !isContainerRunning(pod, containerName) {
				continue
			}
			if retry := w.retries[key]; retry != nil && now.Before(retry.notBefore) {
				continue
			}
			if len(w.active) >= maxPerStep {
				w.incomplete = true
				w.setWarning(key, fmt.Sprintf("Not counting the logs of pod %s container %s: the logs of at most %d containers are followed per step.", pod.Name, containerName, maxPerStep))
				continue
			}
			if !acquireLogStreamSlot(w.executionId) {
				w.incomplete = true
				w.setWarning(key, fmt.Sprintf("Not counting the logs of pod %s container %s: the limit of %d concurrently followed containers is reached.", pod.Name, containerName, extconfig.Config.LogErrorCountMaxConcurrentStreams))
				continue
			}
			w.clearWarning(key)
			cursor, ok := w.cursors[key]
			if !ok {
				// Crashes before the step started are not of interest, so start with the current restart count.
				cursor = &ContainerLogCursor{Pod: pod.Name, Container: containerName, LastTime: w.start, RestartCount: restartCount(pod, containerName)}
				w.cursors[key] = cursor
			}
			w.active[key] = true
			w.wg.Add(1)
			go w.follow(key, cursor, restartCount(pod, containerName))
		}
	}
}

// setWarning and clearWarning must be called holding w.mu.
func (w *logWatcher) setWarning(key, warning string) {
	w.warnings[key] = warning
}

func (w *logWatcher) clearWarning(key string) {
	delete(w.warnings, key)
	// a warning showing up again later is reported again
	delete(w.reportedWarnings, key)
}

func (w *logWatcher) follow(key string, cursor *ContainerLogCursor, currentRestartCount int32) {
	defer w.wg.Done()
	defer releaseLogStreamSlot(w.executionId)
	defer func() {
		w.mu.Lock()
		delete(w.active, key)
		w.mu.Unlock()
	}()

	w.mu.Lock()
	previousInstanceMissed := currentRestartCount > cursor.RestartCount
	w.mu.Unlock()
	if previousInstanceMissed {
		// The container restarted while it wasn't followed: count what the crashed instance logged before it died.
		if err := w.read(cursor, true); err != nil {
			log.Debug().Err(err).Msgf("Failed to read previous logs of %s/%s container %s", w.namespace, cursor.Pod, cursor.Container)
		}
		w.mu.Lock()
		cursor.RestartCount = currentRestartCount
		w.mu.Unlock()
	}

	err := w.read(cursor, false)
	w.mu.Lock()
	defer w.mu.Unlock()
	if err == nil || w.streamsCtx.Err() != nil {
		delete(w.retries, key)
		return
	}
	log.Debug().Err(err).Msgf("Failed to follow logs of %s/%s container %s", w.namespace, cursor.Pod, cursor.Container)
	retry := w.retries[key]
	if retry == nil {
		retry = &logStreamRetry{}
		w.retries[key] = retry
	}
	retry.failures++
	backoff := min(logStreamRetryMinBackoff<<(min(retry.failures, 5)-1), logStreamRetryMaxBackoff)
	retry.notBefore = time.Now().Add(backoff)
	w.setWarning(key, fmt.Sprintf("Failed to read logs of pod %s container %s: %s", cursor.Pod, cursor.Container, err.Error()))
}

func (w *logWatcher) read(cursor *ContainerLogCursor, previous bool) error {
	w.mu.Lock()
	sinceTime := metav1.NewTime(cursor.LastTime)
	w.mu.Unlock()
	stream, err := w.streamer.PodLogs(w.streamsCtx, w.namespace, cursor.Pod, &corev1.PodLogOptions{
		Container:  cursor.Container,
		Previous:   previous,
		Follow:     !previous,
		Timestamps: true,
		SinceTime:  &sinceTime,
	})
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()
	err = countLogLines(&countingReader{reader: stream, count: &w.bytesRead}, cursor, w.errorMatcher, w.ignoreRegex, w.end, &w.mu)
	if w.streamsCtx.Err() != nil {
		// closed on purpose
		return nil
	}
	return err
}

func (w *logWatcher) isBudgetExceeded() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.budgetExceeded
}

func (w *logWatcher) isIncomplete() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.incomplete
}

// setWorkloadMissing reports (once) that the workload doesn't exist anymore.
func (w *logWatcher) setWorkloadMissing(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err == nil {
		w.clearWarning(workloadMissingWarningKey)
		return
	}
	w.setWarning(workloadMissingWarningKey, fmt.Sprintf("Counting only the logs read so far: %s", err.Error()))
}

// exceedBudget closes the streams of the watcher for good. The counts read so far are kept.
func (w *logWatcher) exceedBudget(stepRate, totalRate int64) {
	w.mu.Lock()
	w.budgetExceeded = true
	w.incomplete = true
	w.warnings[logVolumeBudgetWarningKey] = fmt.Sprintf("Stopped counting the logs of %s: the log volume budget of %d MiB/s for all running Error Log Count checks was exceeded (measured %.1f MiB/s, of which this step %.1f MiB/s). The counts up to now are kept.", w.namespace+"/"+w.target, extconfig.Config.LogErrorCountMaxLogMiBPerSecond, float64(totalRate)/(1024*1024), float64(stepRate)/(1024*1024))
	w.mu.Unlock()
	w.cancelStreams()
}

// wait blocks until all streams have ended (tests only).
func (w *logWatcher) wait() {
	w.wg.Wait()
}

// snapshot returns a copy of the cursors, to be stored in the action state.
func (w *logWatcher) snapshot() map[string]*ContainerLogCursor {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make(map[string]*ContainerLogCursor, len(w.cursors))
	for key, cursor := range w.cursors {
		c := *cursor
		result[key] = &c
	}
	return result
}

// drainWarnings returns the warnings that changed since the last call.
func (w *logWatcher) drainWarnings() []action_kit_api.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	var messages []action_kit_api.Message
	for key, warning := range w.warnings {
		if w.reportedWarnings[key] == warning {
			continue
		}
		w.reportedWarnings[key] = warning
		messages = append(messages, action_kit_api.Message{
			Level:   extutil.Ptr(action_kit_api.Warn),
			Message: warning,
		})
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].Message < messages[j].Message })
	return messages
}

const (
	logVolumeWindowSeconds    = 10
	logVolumeBudgetWarningKey = "log-volume-budget"
	workloadMissingWarningKey = "workload-missing"
)

type countingReader struct {
	reader io.Reader
	count  *atomic.Int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.count.Add(int64(n))
	return n, err
}

var logVolumeMonitorOnce sync.Once

// startLogVolumeMonitor samples the log volume of all watchers once per second and enforces the budget.
func startLogVolumeMonitor() {
	logVolumeMonitorOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				enforceLogVolumeBudget()
			}
		}()
	})
}

// enforceLogVolumeBudget takes a one second sample of the bytes read by every watcher. If the volume of all watchers,
// averaged over the last logVolumeWindowSeconds, exceeds the budget, the streams of the steps reading the most are
// closed until the remaining volume fits into the budget.
func enforceLogVolumeBudget() {
	logWatchers.Lock()
	defer logWatchers.Unlock()

	type stepVolume struct {
		watcher *logWatcher
		bytes   int64
	}
	var steps []stepVolume
	total := int64(0)
	for _, w := range logWatchers.m {
		w.volumeWindow[w.volumeIndex] = w.bytesRead.Swap(0)
		w.volumeIndex = (w.volumeIndex + 1) % logVolumeWindowSeconds
		sum := int64(0)
		for _, b := range w.volumeWindow {
			sum += b
		}
		if w.isBudgetExceeded() {
			continue
		}
		steps = append(steps, stepVolume{watcher: w, bytes: sum})
		total += sum
	}

	budget := int64(extconfig.Config.LogErrorCountMaxLogMiBPerSecond) * 1024 * 1024 * logVolumeWindowSeconds
	if budget <= 0 || total <= budget {
		return
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].bytes > steps[j].bytes })
	measured := total
	for _, step := range steps {
		if total <= budget {
			break
		}
		log.Warn().Msgf("Log volume budget exceeded (%d bytes in %ds); closing the log streams of %s/%s", measured, logVolumeWindowSeconds, step.watcher.namespace, step.watcher.target)
		step.watcher.exceedBudget(step.bytes/logVolumeWindowSeconds, measured/logVolumeWindowSeconds)
		total -= step.bytes
	}
}
