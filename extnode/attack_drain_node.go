// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

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

type DrainNodeAction struct{}

type DrainNodeState struct {
	Node             string `json:"node"`
	CmdStateID       string `json:"cmdStateId"`
	Pid              int    `json:"pid"`
	CommandCompleted bool   `json:"commandCompleted"`
}

func NewDrainNodeAction() action_kit_sdk.Action[DrainNodeState] {
	return DrainNodeAction{}
}

var _ action_kit_sdk.Action[DrainNodeState] = (*DrainNodeAction)(nil)
var _ action_kit_sdk.ActionWithStatus[DrainNodeState] = (*DrainNodeAction)(nil)
var _ action_kit_sdk.ActionWithStop[DrainNodeState] = (*DrainNodeAction)(nil)

func (f DrainNodeAction) NewEmptyState() DrainNodeState {
	return DrainNodeState{}
}

func (f DrainNodeAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          DrainNodeActionId,
		Label:       "Drain Node",
		Description: "Drain a Kubernetes node",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(drainNodeIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: NodeTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find node by its name"),
					Query:       "host.hostname=\"\"",
				},
			}),
		}),
		Category:    extutil.Ptr("state"),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Name:         "duration",
				Type:         action_kit_api.Duration,
				Description:  extutil.Ptr("The duration of the attack. The node will be uncordoned after the attack."),
				Advanced:     extutil.Ptr(false),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("180s"),
				Order:        extutil.Ptr(0),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/node/attack/drain-node/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/node/attack/drain-node/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method: "POST",
			Path:   "/node/attack/drain-node/status",
		}),
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/node/attack/drain-node/stop",
		}),
	}
}

func (f DrainNodeAction) Prepare(_ context.Context, state *DrainNodeState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.Node = request.Target.Attributes["host.hostname"][0]
	return nil, nil
}

func (f DrainNodeAction) Start(_ context.Context, state *DrainNodeState) (*action_kit_api.StartResult, error) {
	command := []string{"kubectl",
		"drain",
		state.Node,
		"--pod-selector=steadybit.com/extension!=true,steadybit.com/outpost!=true,steadybit.com/agent!=true",
		"--delete-emptydir-data",
		"--ignore-daemonsets",
		"--force"}

	log.Info().Str("node", state.Node).Msgf("Drain Node with command '%s'", strings.Join(command, " "))

	cmd := exec.Command(command[0], command[1:]...)

	cmdState := extcmd.NewCmdState(cmd)
	state.CmdStateID = cmdState.Id
	err := cmd.Start()
	if err != nil {
		return nil, extension_kit.ToError("Failed to drain node.", err)
	}

	state.Pid = cmd.Process.Pid
	go func() {
		cmdErr := cmd.Wait()
		if cmdErr != nil {
			log.Error().Str("node", state.Node).Msgf("Failed to drain node: %s", cmdErr)
		}
	}()

	return nil, nil
}
func (f DrainNodeAction) Status(_ context.Context, state *DrainNodeState) (*action_kit_api.StatusResult, error) {
	log.Debug().Int("pid", state.Pid).Str("node", state.Node).Msgf("Checking command...")

	cmdState, err := extcmd.GetCmdState(state.CmdStateID)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find command state", err)
	}

	var result action_kit_api.StatusResult

	if !state.CommandCompleted {
		messages := make([]action_kit_api.Message, 0)
		// check if drain node command is still running
		exitCode := cmdState.Cmd.ProcessState.ExitCode()
		stdOut := cmdState.GetLines(false)
		stdOutToLog(stdOut, state.Node)
		if exitCode == -1 {
			log.Debug().Str("node", state.Node).Msgf("Drain node still running")
			messages = append(messages, action_kit_api.Message{
				Level:   extutil.Ptr(action_kit_api.Debug),
				Message: fmt.Sprintf("Drain node '%s' still running", state.Node),
			})
		} else if exitCode == 0 {
			log.Info().Str("node", state.Node).Msgf("Drain node completed successfully")
			messages = append(messages, action_kit_api.Message{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("Drain node '%s' completed successfully", state.Node),
			})
			state.CommandCompleted = true
		} else {
			title := fmt.Sprintf("Failed to drain node, exit-code %d", exitCode)
			stdOutError := extractErrorFromStdOut(stdOut)
			if stdOutError != nil {
				title = *stdOutError
			}
			result.Completed = true
			result.Error = &action_kit_api.ActionKitError{
				Status: extutil.Ptr(action_kit_api.Errored),
				Title:  title,
			}
			state.CommandCompleted = true
		}
		result.Messages = extutil.Ptr(messages)
	}

	return &result, nil
}

func stdOutToLog(lines []string, node string) {
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ReplaceAll(line, "\n", ""))
		if len(trimmed) > 0 {
			log.Info().Str("node", node).Msgf("---- %s", trimmed)
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

func (f DrainNodeAction) Stop(_ context.Context, state *DrainNodeState) (*action_kit_api.StopResult, error) {
	if state.CmdStateID == "" {
		log.Debug().Str("node", state.Node).Msg("Command not yet started, nothing to stop.")
		return nil, nil
	}

	if !state.CommandCompleted {
		// kill drain command if it is still running
		var pid = state.Pid
		process, err := os.FindProcess(pid)
		if err != nil {
			return nil, extension_kit.ToError("Failed to find process", err)
		}
		_ = process.Kill()
		log.Debug().Str("node", state.Node).Msg("Drain node command was still running - killed now.")
	}

	// remove cmd state and read remaining stdout
	cmdState, err := extcmd.GetCmdState(state.CmdStateID)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find command state", err)
	}
	extcmd.RemoveCmdState(state.CmdStateID)

	// read Stout and log it
	stdOut := cmdState.GetLines(true)
	stdOutToLog(stdOut, state.Node)

	// uncordon node
	cmd := exec.Command("kubectl", "uncordon", state.Node)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to uncordon node: %s", string(output)), err)
	}
	log.Info().Str("node", state.Node).Msgf("Node uncordoned")

	messages := make([]action_kit_api.Message, 0)
	messages = append(messages, action_kit_api.Message{
		Level:   extutil.Ptr(action_kit_api.Info),
		Message: fmt.Sprintf("Drain node '%s' successfully stopped and uncordoned.", state.Node),
	})

	return &action_kit_api.StopResult{
		Messages: extutil.Ptr(messages),
	}, nil
}
