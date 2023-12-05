package extcommon

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extcmd"
	"github.com/steadybit/extension-kit/extutil"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"os/exec"
	"strings"
)

// Base for actions executing a kubectl-command in the background, checking the state periodically and stopping the command and optionally rolling it back with another command.
// - if the action defines a duration, the action continues to run until the duration is over
// - if the action does not define a duration, the action is stopped after the command has completed

type KubectlOpts struct {
	Command         []string  `json:"command"`
	RollbackCommand *[]string `json:"rollbackCommand,omitempty"`
	LogTargetType   string    `json:"targetType"`
	LogTargetName   string    `json:"targetName"`
	LogActionName   string    `json:"actionName"`
}

type KubectlActionState struct {
	Opts             KubectlOpts `json:"opts"`
	CmdStateID       string      `json:"cmdStateId"`
	Pid              int         `json:"pid"`
	CommandCompleted bool        `json:"commandCompleted"`
}

type KubectlOptsProvider func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*KubectlOpts, error)

type KubectlAction struct {
	Description  action_kit_api.ActionDescription
	OptsProvider KubectlOptsProvider
}

var _ action_kit_sdk.Action[KubectlActionState] = (*KubectlAction)(nil)
var _ action_kit_sdk.ActionWithStatus[KubectlActionState] = (*KubectlAction)(nil)
var _ action_kit_sdk.ActionWithStop[KubectlActionState] = (*KubectlAction)(nil)

func (a KubectlAction) NewEmptyState() KubectlActionState {
	return KubectlActionState{}
}

func (a KubectlAction) Describe() action_kit_api.ActionDescription {
	return a.Description
}

func (a KubectlAction) Prepare(ctx context.Context, state *KubectlActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	opts, err := a.OptsProvider(ctx, request)
	if err != nil {
		extensionError, isExtensionError := err.(extension_kit.ExtensionError)
		if isExtensionError {
			return nil, extensionError
		} else {
			return nil, extension_kit.ToError("Failed to prepare settings.", err)
		}
	}
	state.Opts = *opts
	return nil, nil
}

func (a KubectlAction) Start(_ context.Context, state *KubectlActionState) (*action_kit_api.StartResult, error) {
	log.Info().
		Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
		Msgf("%s with command '%s'", cases.Title(language.Und).String(state.Opts.LogActionName), strings.Join(state.Opts.Command, " "))
	cmd := exec.Command(state.Opts.Command[0], state.Opts.Command[1:]...)

	cmdState := extcmd.NewCmdState(cmd)
	state.CmdStateID = cmdState.Id
	err := cmd.Start()
	if err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to %s.", state.Opts.LogActionName), err)
	}

	state.Pid = cmd.Process.Pid
	go func() {
		cmdErr := cmd.Wait()
		if cmdErr != nil {
			log.Error().
				Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
				Msgf("Failed to %s: %s", state.Opts.LogActionName, cmdErr)
		}
	}()

	return nil, nil
}

func (a KubectlAction) Status(_ context.Context, state *KubectlActionState) (*action_kit_api.StatusResult, error) {
	var result action_kit_api.StatusResult

	if !state.CommandCompleted {
		log.Debug().
			Int("pid", state.Pid).
			Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
			Msgf("Checking command...")

		cmdState, err := extcmd.GetCmdState(state.CmdStateID)
		if err != nil {
			return nil, extension_kit.ToError("Failed to find command state", err)
		}

		messages := make([]action_kit_api.Message, 0)
		// check if command is still running
		exitCode := cmdState.Cmd.ProcessState.ExitCode()
		stdOut := cmdState.GetLines(false)
		stdOutToLog(stdOut, state.Opts)
		if exitCode == -1 {
			log.Debug().
				Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
				Msgf("%s still running", cases.Title(language.Und).String(state.Opts.LogActionName))
			messages = append(messages, action_kit_api.Message{
				Level:   extutil.Ptr(action_kit_api.Debug),
				Message: fmt.Sprintf("%s '%s' still running", cases.Title(language.Und).String(state.Opts.LogActionName), state.Opts.LogTargetName),
			})
		} else if exitCode == 0 {
			log.Info().
				Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
				Msgf("%s completed successfully", state.Opts.LogActionName)
			messages = append(messages, action_kit_api.Message{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("%s '%s' completed successfully", cases.Title(language.Und).String(state.Opts.LogActionName), state.Opts.LogTargetName),
			})
			state.CommandCompleted = true
			if !hasDuration(&a.Description) {
				result.Completed = true
			}
		} else {
			title := fmt.Sprintf("Failed to %s exit-code %d", state.Opts.LogActionName, exitCode)
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

func stdOutToLog(lines []string, opts KubectlOpts) {
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ReplaceAll(line, "\n", ""))
		if len(trimmed) > 0 {
			log.Info().
				Str(opts.LogTargetType, opts.LogTargetName).
				Msgf("---- %s", trimmed)
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

func hasDuration(description *action_kit_api.ActionDescription) bool {
	for _, param := range (*description).Parameters {
		if param.Name == "duration" {
			return true
		}
	}
	return false
}

func (a KubectlAction) Stop(_ context.Context, state *KubectlActionState) (*action_kit_api.StopResult, error) {
	if state.CmdStateID == "" {
		log.Debug().
			Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
			Msg("Command not yet started, nothing to stop.")
		return nil, nil
	}

	if !state.CommandCompleted {
		// kill command if it is still running
		var pid = state.Pid
		process, err := os.FindProcess(pid)
		if err != nil {
			return nil, extension_kit.ToError("Failed to find process", err)
		}
		_ = process.Kill()
		log.Debug().
			Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
			Msg("Command was still running - killed now.")
	}

	// remove cmd state and read remaining stdout
	cmdState, err := extcmd.GetCmdState(state.CmdStateID)
	if err == nil {
		extcmd.RemoveCmdState(state.CmdStateID)
		// read Stout and log it
		stdOut := cmdState.GetLines(true)
		stdOutToLog(stdOut, state.Opts)
	}

	// rollback action
	if state.Opts.RollbackCommand != nil {
		log.Info().
			Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
			Msgf("Rollback %s with command '%s'", state.Opts.LogActionName, strings.Join(*state.Opts.RollbackCommand, " "))

		cmd := exec.Command((*state.Opts.RollbackCommand)[0], (*state.Opts.RollbackCommand)[1:]...)
		output, rollbackErr := cmd.CombinedOutput()

		if rollbackErr != nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to rollback %s: %s", state.Opts.LogActionName, string(output)), rollbackErr)
		}
		log.Debug().
			Str(state.Opts.LogTargetType, state.Opts.LogTargetName).
			Msgf("Rollback completed.")
	}

	messages := make([]action_kit_api.Message, 0)
	messages = append(messages, action_kit_api.Message{
		Level:   extutil.Ptr(action_kit_api.Info),
		Message: fmt.Sprintf("%s '%s' successfully stopped.", cases.Title(language.Und).String(state.Opts.LogActionName), state.Opts.LogTargetName),
	})

	return &action_kit_api.StopResult{
		Messages: extutil.Ptr(messages),
	}, nil
}
