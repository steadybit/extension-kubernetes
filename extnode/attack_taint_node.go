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
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"os"
	"os/exec"
	"strings"
)

type TaintNodeAction struct{}

type TaintNodeState struct {
	Taint            string `json:"taint"`
	Node             string `json:"node"`
	CmdStateID       string `json:"cmdStateId"`
	Pid              int    `json:"pid"`
	CommandCompleted bool   `json:"commandCompleted"`
}

type TaintNodeConfig struct {
	Key    string
	Value  string
	Effect string
}

func NewTaintNodeAction() action_kit_sdk.Action[TaintNodeState] {
	return TaintNodeAction{}
}

var _ action_kit_sdk.Action[TaintNodeState] = (*TaintNodeAction)(nil)
var _ action_kit_sdk.ActionWithStatus[TaintNodeState] = (*TaintNodeAction)(nil)
var _ action_kit_sdk.ActionWithStop[TaintNodeState] = (*TaintNodeAction)(nil)

func (f TaintNodeAction) NewEmptyState() TaintNodeState {
	return TaintNodeState{}
}

func (f TaintNodeAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          TaintNodeActionId,
		Label:       "Taint Node",
		Description: "Taint a Kubernetes node",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(taintNodeIcon),
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
				Description:  extutil.Ptr("The duration of the attack. The taint will be removed after the attack."),
				Advanced:     extutil.Ptr(false),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("180s"),
				Order:        extutil.Ptr(0),
			},
			{
				Label:       "Key",
				Name:        "key",
				Type:        action_kit_api.String,
				Description: extutil.Ptr("The key of the taint."),
				Advanced:    extutil.Ptr(false),
				Required:    extutil.Ptr(true),
				Order:       extutil.Ptr(1),
			},
			{
				Label:       "Value",
				Name:        "value",
				Type:        action_kit_api.String,
				Description: extutil.Ptr("The optional value of the taint."),
				Advanced:    extutil.Ptr(false),
				Required:    extutil.Ptr(false),
				Order:       extutil.Ptr(1),
			},
			{
				Label:        "Effect",
				Name:         "effect",
				Type:         action_kit_api.String,
				Description:  extutil.Ptr("The effect of the taint."),
				Advanced:     extutil.Ptr(false),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("NoSchedule"),
				Order:        extutil.Ptr(2),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "NoSchedule",
						Value: "NoSchedule",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "PreferNoSchedule",
						Value: "PreferNoSchedule",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "NoExecute",
						Value: "NoExecute",
					},
				}),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:    &action_kit_api.MutatingEndpointReference{},
	}
}

func (f TaintNodeAction) Prepare(_ context.Context, state *TaintNodeState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config TaintNodeConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}

	taint := config.Key
	if config.Value != "" {
		taint = taint + "=" + config.Value
	}
	taint = taint + ":" + config.Effect

	state.Taint = taint
	state.Node = request.Target.Attributes["host.hostname"][0]
	return nil, nil
}

func (f TaintNodeAction) Start(_ context.Context, state *TaintNodeState) (*action_kit_api.StartResult, error) {
	command := []string{"kubectl",
		"taint",
		"node",
		state.Node,
		state.Taint}

	log.Info().Str("node", state.Node).Msgf("Taint Node with command '%s'", strings.Join(command, " "))

	cmd := exec.Command(command[0], command[1:]...)

	cmdState := extcmd.NewCmdState(cmd)
	state.CmdStateID = cmdState.Id
	err := cmd.Start()
	if err != nil {
		return nil, extension_kit.ToError("Failed to taint node.", err)
	}

	state.Pid = cmd.Process.Pid
	go func() {
		cmdErr := cmd.Wait()
		if cmdErr != nil {
			log.Error().Str("node", state.Node).Msgf("Failed to taint node: %s", cmdErr)
		}
	}()

	return nil, nil
}
func (f TaintNodeAction) Status(_ context.Context, state *TaintNodeState) (*action_kit_api.StatusResult, error) {
	log.Debug().Int("pid", state.Pid).Str("node", state.Node).Msgf("Checking command...")

	cmdState, err := extcmd.GetCmdState(state.CmdStateID)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find command state", err)
	}

	var result action_kit_api.StatusResult

	if !state.CommandCompleted {
		messages := make([]action_kit_api.Message, 0)
		// check if taint node command is still running
		exitCode := cmdState.Cmd.ProcessState.ExitCode()
		stdOut := cmdState.GetLines(false)
		stdOutToLog(stdOut, state.Node)
		if exitCode == -1 {
			log.Debug().Str("node", state.Node).Msgf("Taint node still running")
			messages = append(messages, action_kit_api.Message{
				Level:   extutil.Ptr(action_kit_api.Debug),
				Message: fmt.Sprintf("Taint node '%s' still running", state.Node),
			})
		} else if exitCode == 0 {
			log.Info().Str("node", state.Node).Msgf("Taint node completed successfully")
			messages = append(messages, action_kit_api.Message{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("Taint node '%s' completed successfully", state.Node),
			})
			state.CommandCompleted = true
		} else {
			title := fmt.Sprintf("Failed to taint node, exit-code %d", exitCode)
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

func (f TaintNodeAction) Stop(_ context.Context, state *TaintNodeState) (*action_kit_api.StopResult, error) {
	if state.CmdStateID == "" {
		log.Debug().Str("node", state.Node).Msg("Command not yet started, nothing to stop.")
		return nil, nil
	}

	if !state.CommandCompleted {
		// kill taint command if it is still running
		var pid = state.Pid
		process, err := os.FindProcess(pid)
		if err != nil {
			return nil, extension_kit.ToError("Failed to find process", err)
		}
		_ = process.Kill()
		log.Debug().Str("node", state.Node).Msg("Taint node command was still running - killed now.")
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

	// remove node
	cmd := exec.Command("kubectl", "taint", "node", state.Node, state.Taint+"-")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to untaint node: %s", string(output)), err)
	}
	log.Info().Str("node", state.Node).Msgf("Node untainted")

	messages := make([]action_kit_api.Message, 0)
	messages = append(messages, action_kit_api.Message{
		Level:   extutil.Ptr(action_kit_api.Info),
		Message: fmt.Sprintf("Taint node '%s' successfully stopped and untainted.", state.Node),
	})

	return &action_kit_api.StopResult{
		Messages: extutil.Ptr(messages),
	}, nil
}
