// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpod

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extcmd"
	"github.com/steadybit/extension-kit/extutil"
	"os"
	"os/exec"
	"strings"
)

type DeletePodAction struct {
}

type DeletePodActionState struct {
	Namespace  string `json:"namespace"`
	Pod        string `json:"pod"`
	CmdStateID string `json:"cmdStateId"`
	Pid        int    `json:"pid"`
}

func NewDeletePodAction() action_kit_sdk.Action[DeletePodActionState] {
	return DeletePodAction{}
}

var _ action_kit_sdk.Action[DeletePodActionState] = (*DeletePodAction)(nil)
var _ action_kit_sdk.ActionWithStatus[DeletePodActionState] = (*DeletePodAction)(nil)
var _ action_kit_sdk.ActionWithStop[DeletePodActionState] = (*DeletePodAction)(nil)

func (f DeletePodAction) NewEmptyState() DeletePodActionState {
	return DeletePodActionState{}
}

func (f DeletePodAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          DeletePodActionId,
		Label:       "Delete Pod",
		Description: "Delete Pods in a Kubernetes cluster",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(deletePodActionIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: PodTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find pods by cluster, namespace and deployment"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlInternal,
		Kind:        action_kit_api.Attack,
		Parameters:  []action_kit_api.ActionParameter{},
		Prepare:     action_kit_api.MutatingEndpointReference{},
		Start:       action_kit_api.MutatingEndpointReference{},
		Status:      &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:        &action_kit_api.MutatingEndpointReference{},
	}
}

func (f DeletePodAction) Prepare(_ context.Context, state *DeletePodActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.Pod = request.Target.Attributes["k8s.pod.name"][0]
	return nil, nil
}

func (f DeletePodAction) Start(_ context.Context, state *DeletePodActionState) (*action_kit_api.StartResult, error) {
	command := []string{"kubectl",
		"delete",
		"pod",
		"--namespace",
		state.Namespace,
		state.Pod}

	log.Info().Str("pod", state.Pod).Msgf("Delete pod with command '%s'", strings.Join(command, " "))

	cmd := exec.Command(command[0], command[1:]...)

	cmdState := extcmd.NewCmdState(cmd)
	state.CmdStateID = cmdState.Id
	err := cmd.Start()
	if err != nil {
		return nil, extension_kit.ToError("Failed to delete pod.", err)
	}

	state.Pid = cmd.Process.Pid
	go func() {
		cmdErr := cmd.Wait()
		if cmdErr != nil {
			log.Error().Str("pod", state.Pod).Msgf("Failed to delete pod: %s", cmdErr)
		}
	}()

	return nil, nil
}

func (f DeletePodAction) Status(_ context.Context, state *DeletePodActionState) (*action_kit_api.StatusResult, error) {
	log.Debug().Int("pid", state.Pid).Str("pod", state.Pod).Msgf("Checking command...")

	cmdState, err := extcmd.GetCmdState(state.CmdStateID)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find command state", err)
	}

	var result action_kit_api.StatusResult

	messages := make([]action_kit_api.Message, 0)
	// check if command is still running
	exitCode := cmdState.Cmd.ProcessState.ExitCode()
	stdOut := cmdState.GetLines(false)
	stdOutToLog(stdOut, state.Pod)
	if exitCode == -1 {
		log.Debug().Str("pod", state.Pod).Msgf("Delete Pod still running")
		messages = append(messages, action_kit_api.Message{
			Level:   extutil.Ptr(action_kit_api.Debug),
			Message: fmt.Sprintf("Delete Pod '%s' still running", state.Pod),
		})
	} else if exitCode == 0 {
		log.Debug().Str("pod", state.Pod).Msgf("Delete Pod completed successfully")
		messages = append(messages, action_kit_api.Message{
			Level:   extutil.Ptr(action_kit_api.Info),
			Message: fmt.Sprintf("Delete Pod '%s' completed successfully", state.Pod),
		})
		result.Completed = true
	} else {
		title := fmt.Sprintf("Failed to delete pod, exit-code %d", exitCode)
		stdOutError := extractErrorFromStdOut(stdOut)
		if stdOutError != nil {
			title = *stdOutError
		}
		result.Completed = true
		result.Error = &action_kit_api.ActionKitError{
			Status: extutil.Ptr(action_kit_api.Errored),
			Title:  title,
		}
		result.Completed = true
	}
	result.Messages = extutil.Ptr(messages)
	return &result, nil
}

func stdOutToLog(lines []string, pod string) {
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ReplaceAll(line, "\n", ""))
		if len(trimmed) > 0 {
			log.Info().Str("pod", pod).Msgf("---- %s", trimmed)
		}
	}
}

func extractErrorFromStdOut(lines []string) *string {
	//Find error, last log lines first
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "error: ") {
			split := strings.SplitAfter(lines[i], "error: ")
			if len(split) > 1 {
				return extutil.Ptr(strings.Join(split[1:], ""))
			}
		}
	}
	return nil
}

func (f DeletePodAction) Stop(_ context.Context, state *DeletePodActionState) (*action_kit_api.StopResult, error) {
	if state.CmdStateID == "" {
		log.Debug().Str("pod", state.Pod).Msg("Command not yet started, nothing to stop.")
		return nil, nil
	}

	// kill drain command if it is still running
	var pid = state.Pid
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find process", err)
	}
	_ = process.Kill()
	log.Debug().Str("pod", state.Pod).Msg("Delete pod command was still running - killed now.")

	// remove cmd state and read remaining stdout
	cmdState, err := extcmd.GetCmdState(state.CmdStateID)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find command state", err)
	}
	extcmd.RemoveCmdState(state.CmdStateID)

	// read Stout and log it
	stdOut := cmdState.GetLines(true)
	stdOutToLog(stdOut, state.Pod)

	messages := make([]action_kit_api.Message, 0)
	messages = append(messages, action_kit_api.Message{
		Level:   extutil.Ptr(action_kit_api.Info),
		Message: fmt.Sprintf("Delete pod '%s' successfully stopped.", state.Pod),
	})

	return &action_kit_api.StopResult{
		Messages: extutil.Ptr(messages),
	}, nil
}
