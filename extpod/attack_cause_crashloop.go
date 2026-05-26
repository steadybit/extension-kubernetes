// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpod

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kubernetes/v2/client"
)

type CrashLoopAction struct {
}

type CrashLoopState struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container,omitempty"`
	Signal    string `json:"signal"`
}

type CrashLoopConfig struct {
	Container string `json:"container,omitempty"`
	Signal    string `json:"signal,omitempty"`
}

func NewCrashLoopAction() action_kit_sdk.Action[CrashLoopState] {
	return CrashLoopAction{}
}

var _ action_kit_sdk.Action[CrashLoopState] = (*CrashLoopAction)(nil)
var _ action_kit_sdk.ActionWithStatus[CrashLoopState] = (*CrashLoopAction)(nil)

func (f CrashLoopAction) NewEmptyState() CrashLoopState {
	return CrashLoopState{}
}

func (f CrashLoopAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:              CrashLoopActionId,
		Label:           "Cause Crash Loop",
		Description:     "Cause the containers of a pod to crash in a loop",
		Version:         extbuild.GetSemverVersionStringOrUnknown(),
		Icon:            new("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xNy44NDk2IDYuNzgwMDJDMTguMTE1NyA2Ljk0MTU4IDE4LjI0ODcgNy4yNDU2OCAxOC4yMTA3IDcuNTMwNzlMMTguMjIwMiA3LjU0MDI5QzE4LjI0ODcgNy44NDQ0IDE4LjEwNjEgOC4xNDg1MSAxNy44MjEgOC4zMDA1NkwxNC4xMTQ3IDEwLjI0ODhDMTQuMDAwNyAxMC4zMDU4IDEzLjg4NjYgMTAuMzM0MyAxMy43NjMxIDEwLjMzNDNDMTMuNDg3NSAxMC4zMzQzIDEzLjIyMTQgMTAuMTgyMiAxMy4wODgzIDkuOTI1NjVDMTIuODg4OCA5LjU1NTAxIDEzLjAzMTMgOS4wOTg4NSAxMy40MDIgOC44OTkyOEwxNS45OTY0IDcuNTMwNzlMMTAuMTcwOCA0LjY4OTI3QzkuODQ3NjggNC40NzA2OSA5LjM4MjAxIDQuNDgwMTkgOC45NzMzNyA0LjcxNzc4TDMuMzg1MzYgNy41NTkzQzQuNjM5ODEgOC4yMzQwNCA3Ljg4MDQ3IDkuOTczMTYgOS4xNjM0MyAxMC40NzY4QzkuNDg2NTUgMTAuNjA5OSA5Ljc2MjE1IDEwLjYxOTQgOS43NjIxNSAxMC42MTk0QzEwLjE3MDggMTAuNjI4OSAxMC41MDM0IDEwLjk4MDUgMTAuNDkzOSAxMS4zOTg3VjExLjQzNjdWMjEuMzQ4OEMxMC40OTM5IDIxLjY4MTQgMTAuMjc1MyAyMS45NTcgOS45NzEyMiAyMi4wNTJDOS44NjY2OSAyMi4wOTk1IDkuNzYyMTUgMjIuMTM3NSA5LjYzODYgMjIuMTM3NUM5LjE5MTk0IDIyLjEzNzUgOC43MjYyOCAyMi4wNDI1IDguMzA4MTMgMjEuODYxOUwyLjY1MzU5IDE4Ljk5MTlDMS42NjUyNCAxOC40OTc3IDEgMTcuMzg1OCAxIDE2LjIyNjRWOS4wMDM4MkMxIDcuNzg3MzggMS42MTc3MiA2Ljc1MTUxIDIuNjUzNTkgNi4yMjg4Mkw4LjI1MTExIDMuMzY4MjlDOS4xMTU5MiAyLjg3NDExIDEwLjE2MTMgMi44NzQxMSAxMC45NjkxIDMuMzg3M0wxNi44NDIyIDYuMjQ3ODNDMTcuMDQxOCA2LjM1MjM2IDE3LjE3NDggNi40MTg4OSAxNy4yODg5IDYuNDc1OTFMMTcuMjg4OSA2LjQ3NTkyQzE3LjQ3ODkgNi41NzA5NSAxNy42MzEgNi42NDY5OCAxNy44NDk2IDYuNzgwMDJaTTguOTkyMzcgMjAuNTAyOVYxMi4wMTY0SDguOTgyODdDOC44Njg4MyAxMS45ODc5IDguNzM1NzggMTEuOTQ5OSA4LjYwMjczIDExLjg5MjlDNy4xNzcyMiAxMS4zNDE3IDMuNjg5NDcgOS40NTk5OCAyLjUzMDA1IDguODMyNzVDMi41MzAwNSA4Ljg2MTI2IDIuNTI1MyA4Ljg4OTc4IDIuNTIwNTUgOC45MTgyOUMyLjUxNTc5IDguOTQ2OCAyLjUxMTA0IDguOTc1MzEgMi41MTEwNCA5LjAwMzgyVjE2LjIyNjRDMi41MTEwNCAxNi44MTU2IDIuODUzMTcgMTcuMzk1MyAzLjMyODM0IDE3LjYzMjlMOC45NDQ4NSAyMC40ODM5QzguOTU0MzYgMjAuNTAyOSA4Ljk5MjM3IDIwLjUwMjkgOC45OTIzNyAyMC41MDI5Wk0xOC41MjQzIDEwLjM2MjhDMTguNTI0MyAxMC4wMjA3IDE4LjI0ODcgOS43NTQ1OSAxNy45MTYxIDkuNzU0NTlDMTcuNTc0IDkuNzU0NTkgMTcuMzA3OSAxMC4wMzAyIDE3LjMwNzkgMTAuMzYyOFYxMS41Njk3QzE3LjMwNzkgMTEuOTExOSAxNy41NzQgMTIuMTc4IDE3LjkxNjEgMTIuMTc4QzE4LjI1ODIgMTIuMTc4IDE4LjUyNDMgMTEuOTAyNCAxOC41MjQzIDExLjU2OTdWMTAuMzYyOFpNMTEuNTY3OCAxMC40NDgzQzExLjg0MzQgMTAuMjc3MyAxMi4yMjM1IDEwLjM2MjggMTIuMzk0NiAxMC42NDc5TDEyLjM4NTEgMTAuNjM4NEwxNy4zMDc5IDE4LjYxMThWMTQuNTcyOEMxNy4zMDc5IDE0LjIzMDcgMTcuNTc0IDEzLjk2NDYgMTcuOTE2MSAxMy45NjQ2QzE4LjI1ODIgMTMuOTY0NiAxOC41MjQzIDE0LjI0MDIgMTguNTI0MyAxNC41NzI4VjE4LjY0OThMMjEuNjYwNCAxNC4zMzUyQzIxLjg2IDE0LjA2OTEgMjIuMjQwMSAxNC4wMDI2IDIyLjUwNjIgMTQuMjAyMkMyMi43NzIzIDE0LjQwMTggMjIuODM4OCAxNC43ODE5IDIyLjYzOTMgMTUuMDQ4TDE5LjkyMTMgMTguNzkyM0wyMS44NiAxNy43Mzc1QzIyLjE1NDYgMTcuNTg1NCAyMi41MTU3IDE3LjY4OTkgMjIuNjc3MyAxNy45ODQ2QzIyLjgyOTMgMTguMjc5MiAyMi43MjQ4IDE4LjY0MDMgMjIuNDMwMiAxOC44MDE4TDE5LjEyMyAyMC41OThDMTkuMTEzNSAxOS45NDIzIDE4LjU4MTMgMTkuNDEwMSAxNy45MTYxIDE5LjQxMDFDMTcuMjUwOCAxOS40MTAxIDE2LjcwOTEgMTkuOTUxOCAxNi43MDkxIDIwLjYxN0gxOS4xMjNIMjIuNzQzOEMyMy4wODU5IDIwLjYxNyAyMy4zNTIgMjAuODgzMSAyMy4zNTIgMjEuMjI1MkMyMy4zNTIgMjEuNTY3MyAyMy4wNzY0IDIxLjgzMzQgMjIuNzQzOCAyMS44MzM0SDExLjg4MTRDMTEuNTM5MyAyMS44MzM0IDExLjI3MzIgMjEuNTY3MyAxMS4yNzMyIDIxLjIyNTJDMTEuMjczMiAyMC44ODMxIDExLjU0ODggMjAuNjE3IDExLjg4MTQgMjAuNjE3SDE2LjU4NTZMMTIuMTg1NSAxOC4xMjcxQzExLjg5MDkgMTcuOTU2IDExLjc5NTkgMTcuNTk0OSAxMS45NTc0IDE3LjMwMDNDMTIuMTI4NSAxNy4wMDU3IDEyLjQ4OTYgMTYuOTEwNyAxMi43ODQyIDE3LjA3MjJMMTYuMTEwNCAxOC45NTM5TDExLjM2ODIgMTEuMjc1MUMxMS4xOTcyIDEwLjk5OTUgMTEuMjgyNyAxMC42MTk0IDExLjU2NzggMTAuNDQ4M1oiIGZpbGw9IiMxRDI2MzIiLz4KPC9zdmc+Cg=="),
		Technology:      new("Kubernetes"),
		TargetSelection: new(targetSelectionTemplates),
		TimeControl:     action_kit_api.TimeControlExternal,
		Kind:            action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  new("How long should we cause the crash loop."),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: new("30s"),
				Required:     new(true),
			},
			{
				Label:       "Container",
				Description: new("By default all containers of the selected pods are killed. If you specify a container, only the selected container will be killed."),
				Name:        "container",
				Type:        action_kit_api.ActionParameterTypeString,
				Advanced:    new(true),
			},
			{
				Label:        "Signal",
				Description:  new("By default, the container will be killed by `kill -SIGTERM 1`. You can specify a different signal here."),
				Name:         "signal",
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: new("15"),
				Required:     new(true),
				Advanced:     new(true), Options: new([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "SIGHUP (1)",
						Value: "1",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGINT (2)",
						Value: "2",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGQUIT (3)",
						Value: "3",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGILL (4)",
						Value: "4",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGABRT (6)",
						Value: "6",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGBUS (7)",
						Value: "7",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGFPE (8)",
						Value: "8",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGKILL (9)",
						Value: "9",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGUSR1 (10)",
						Value: "10",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGSEGV (11)",
						Value: "11",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGUSR2 (12)",
						Value: "12",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGPIPE (13)",
						Value: "13",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGALRM (14)",
						Value: "14",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGTERM (15)",
						Value: "15",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGSTOP (19)",
						Value: "19",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SIGTSTP (20)",
						Value: "20",
					},
				}),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status: new(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: new("2s"), //Containers are killed in the status endpoint
		}),
	}
}

func (f CrashLoopAction) Prepare(_ context.Context, state *CrashLoopState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config CrashLoopConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}

	namespace := request.Target.Attributes["k8s.namespace"][0]
	podName := request.Target.Attributes["k8s.pod.name"][0]
	pod := client.K8S.PodByNamespaceAndName(namespace, podName)
	if pod == nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Pod %s not found in namespace %s", podName, namespace), nil)
	}
	if pod.Spec.HostPID {
		return nil, extension_kit.ToError(fmt.Sprintf("Pod %s in namespace %s has hostPID enabled. This is not yet supported", podName, namespace), nil)
	}

	if config.Container != "" {
		containerFound := false
		for _, cs := range pod.Spec.Containers {
			if config.Container == cs.Name {
				containerFound = true
				break
			}
		}
		if !containerFound {
			return nil, extension_kit.ToError(fmt.Sprintf("Container %s not found in pod specification %s", config.Container, podName), nil)
		}
	}

	state.Namespace = namespace
	state.Pod = podName
	state.Container = config.Container
	state.Signal = config.Signal
	return nil, nil
}

func (f CrashLoopAction) Start(_ context.Context, state *CrashLoopState) (*action_kit_api.StartResult, error) {
	_, err := statusInternal(state)
	return nil, err
}

func (f CrashLoopAction) Status(_ context.Context, state *CrashLoopState) (*action_kit_api.StatusResult, error) {
	return statusInternal(state)
}

func statusInternal(state *CrashLoopState) (*action_kit_api.StatusResult, error) {
	pod := client.K8S.PodByNamespaceAndName(state.Namespace, state.Pod)
	if pod == nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Pod %s not found in namespace %s", state.Pod, state.Namespace), nil)
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if state.Container != "" && state.Container != cs.Name {
			continue
		}

		if cs.State.Running == nil {
			continue
		}

		signal := state.Signal
		if signal == "" {
			signal = "15"
		}

		if err := runKubectlExec(state.Namespace, state.Pod, cs.Name, []string{"kill", "-" + signal, "1"}); err != nil {
			log.Info().Err(err).Msgf("Direct kill failed for container %s in pod %s, retrying via /bin/sh", cs.Name, state.Pod)

			if err := runKubectlExec(state.Namespace, state.Pod, cs.Name, []string{"/bin/sh", "-c", fmt.Sprintf("kill -%s 1", signal)}); err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}

func runKubectlExec(namespace, podName, containerName string, kubeExecCmd []string) error {
	cmd := append([]string{"kubectl", "exec", podName, "-c", containerName, "-n", namespace, "--"}, kubeExecCmd...)

	log.Info().Msgf("Killing container %s in pod %s with command '%s'", containerName, podName, strings.Join(cmd, " "))

	if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
		output := string(out)
		if strings.Contains(output, "container not found") {
			log.Debug().Str("container", containerName).Str("pod", podName).Msg("Container not found. Skipping.")
			return nil
		}
		if strings.Contains(output, "container not created") {
			log.Debug().Str("container", containerName).Str("pod", podName).Msg("Container not created. Skipping.")
			return nil
		}
		if strings.Contains(output, "failed to load task") {
			log.Debug().Str("container", containerName).Str("pod", podName).Msg("Failed to load taks. Skipping.")
			return nil
		}
		if strings.Contains(output, "cannot exec in a stopped state") {
			log.Debug().Str("container", containerName).Str("pod", podName).Msg("Cannot exec in a stopped state. Skipping.")
			return nil
		}
		if strings.Contains(output, "cannot exec in a stopped container") {
			log.Debug().Str("container", containerName).Str("pod", podName).Msg("Cannot exec in a stopped container. Skipping.")
			return nil
		}
		if strings.Contains(output, "container is in CONTAINER_EXITED state") {
			log.Debug().Str("container", containerName).Str("pod", podName).Msg("Container is in CONTAINER_EXITED state. Skipping.")
			return nil
		}
		if strings.Contains(output, "task") && strings.Contains(output, "not found") {
			log.Debug().Str("container", containerName).Str("pod", podName).Msg("Task not found. Skipping.")
			return nil
		}

		return fmt.Errorf("kubectl exec failed for container %s in pod %s/%s (the signal may not have been delivered; this can happen if the container is restarting, the node is under memory/PID pressure, or host security policies block the runtime): %w: %s", containerName, namespace, podName, err, out)
	}
	return nil
}
