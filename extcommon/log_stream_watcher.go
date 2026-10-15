// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"sync"
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

// Log streams currently open across all running steps, limited by extconfig.Config.LogErrorCountMaxConcurrentStreams
// to bound the load on the API server and the kubelets.
var openLogStreams = struct {
	sync.Mutex
	count int
}{}

func acquireLogStreamSlot() bool {
	openLogStreams.Lock()
	defer openLogStreams.Unlock()
	if openLogStreams.count >= extconfig.Config.LogErrorCountMaxConcurrentStreams {
		return false
	}
	openLogStreams.count++
	return true
}

func releaseLogStreamSlot() {
	openLogStreams.Lock()
	defer openLogStreams.Unlock()
	openLogStreams.count--
}

func openLogStreamCount() int {
	openLogStreams.Lock()
	defer openLogStreams.Unlock()
	return openLogStreams.count
}

func countContainers(pods []*corev1.Pod, containerName string) int {
	count := 0
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			if containerName == "" || container.Name == containerName {
				count++
			}
		}
	}
	return count
}

// checkLogStreamLimits verifies that a step following the given number of containers stays within the configured limits.
func checkLogStreamLimits(containers int) error {
	maxPerStep := extconfig.Config.LogErrorCountMaxContainersPerStep
	if containers > maxPerStep {
		return extension_kit.ToError(fmt.Sprintf("The target has %d containers, but the logs of at most %d containers can be followed per step. Narrow down the target selection or the container name, or raise the limit via STEADYBIT_EXTENSION_LOG_ERROR_COUNT_MAX_CONTAINERS_PER_STEP.", containers, maxPerStep), nil)
	}
	maxStreams := extconfig.Config.LogErrorCountMaxConcurrentStreams
	if open := openLogStreamCount(); open+containers > maxStreams {
		return extension_kit.ToError(fmt.Sprintf("Following %d more containers would exceed the limit of %d concurrently followed containers (%d already in use by other steps). Raise the limit via STEADYBIT_EXTENSION_LOG_ERROR_COUNT_MAX_CONCURRENT_STREAMS.", containers, maxStreams, open), nil)
	}
	return nil
}

// logWatcher follows the logs of the containers of a single step execution (one `follow=true` stream per container).
type logWatcher struct {
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	streamer      PodLogStreamer
	start         time.Time
	namespace     string
	containerName string
	errorMatcher  lineMatcher
	ignoreRegex   *regexp.Regexp
	cursors       map[string]*ContainerLogCursor
	active        map[string]bool
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
func getOrCreateLogWatcher(state *LogErrorCountCheckState, streamer PodLogStreamer, errorMatcher lineMatcher, ignoreRegex *regexp.Regexp) *logWatcher {
	logWatchers.Lock()
	defer logWatchers.Unlock()
	if w, ok := logWatchers.m[state.ExecutionId]; ok {
		return w
	}
	ctx, cancel := context.WithDeadline(context.Background(), state.End.Add(logStreamGracePeriod))
	w := &logWatcher{
		ctx:              ctx,
		cancel:           cancel,
		streamer:         streamer,
		start:            state.Start,
		namespace:        state.Namespace,
		containerName:    state.ContainerName,
		errorMatcher:     errorMatcher,
		ignoreRegex:      ignoreRegex,
		cursors:          make(map[string]*ContainerLogCursor),
		active:           make(map[string]bool),
		warnings:         make(map[string]string),
		reportedWarnings: make(map[string]string),
	}
	for key, cursor := range state.Cursors {
		c := *cursor
		w.cursors[key] = &c
	}
	logWatchers.m[state.ExecutionId] = w
	// Close the streams when the deadline is reached, even if stop is never called.
	context.AfterFunc(ctx, func() { removeLogWatcher(state.ExecutionId, w) })
	return w
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
}

// reconcile opens a log stream for every running container of the given pods that isn't followed yet.
func (w *logWatcher) reconcile(pods []*corev1.Pod) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx.Err() != nil {
		return
	}
	maxPerStep := extconfig.Config.LogErrorCountMaxContainersPerStep
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			if w.containerName != "" && container.Name != w.containerName {
				continue
			}
			key := pod.Namespace + "/" + pod.Name + "/" + container.Name
			// Only running containers are followed; a stream of a crashed container ends and is reopened once the
			// container runs again.
			if w.active[key] || !isContainerRunning(pod, container.Name) {
				continue
			}
			if len(w.active) >= maxPerStep {
				w.warnings[key] = fmt.Sprintf("Not counting the logs of pod %s container %s: the logs of at most %d containers are followed per step.", pod.Name, container.Name, maxPerStep)
				continue
			}
			if !acquireLogStreamSlot() {
				w.warnings[key] = fmt.Sprintf("Not counting the logs of pod %s container %s: the limit of %d concurrently followed containers is reached.", pod.Name, container.Name, extconfig.Config.LogErrorCountMaxConcurrentStreams)
				continue
			}
			delete(w.warnings, key)
			cursor, ok := w.cursors[key]
			if !ok {
				// Crashes before the step started are not of interest, so start with the current restart count.
				cursor = &ContainerLogCursor{Pod: pod.Name, Container: container.Name, LastTime: w.start, RestartCount: restartCount(pod, container.Name)}
				w.cursors[key] = cursor
			}
			w.active[key] = true
			w.wg.Add(1)
			go w.follow(key, cursor, restartCount(pod, container.Name))
		}
	}
}

func (w *logWatcher) follow(key string, cursor *ContainerLogCursor, currentRestartCount int32) {
	defer w.wg.Done()
	defer releaseLogStreamSlot()
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

	if err := w.read(cursor, false); err != nil && w.ctx.Err() == nil {
		log.Debug().Err(err).Msgf("Failed to follow logs of %s/%s container %s", w.namespace, cursor.Pod, cursor.Container)
		w.mu.Lock()
		w.warnings[key] = fmt.Sprintf("Failed to read logs of pod %s container %s: %s", cursor.Pod, cursor.Container, err.Error())
		w.mu.Unlock()
	}
}

func (w *logWatcher) read(cursor *ContainerLogCursor, previous bool) error {
	w.mu.Lock()
	sinceTime := metav1.NewTime(cursor.LastTime)
	w.mu.Unlock()
	stream, err := w.streamer.PodLogs(w.ctx, w.namespace, cursor.Pod, &corev1.PodLogOptions{
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
	err = countLogLines(stream, cursor, w.errorMatcher, w.ignoreRegex, &w.mu)
	if w.ctx.Err() != nil {
		// closed on purpose
		return nil
	}
	return err
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
