// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/utils"
	"net/http"
	"os/exec"
	"strings"
)

type DeploymentRolloutRestartState struct {
	Cluster    string `json:"cluster"`
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	Wait       bool   `json:"wait"`
}

func RegisterDeploymentRolloutRestartAttackHandlers() {
	exthttp.RegisterHttpHandler("/deployment/attack/rollout-restart", exthttp.GetterAsHandler(getDeploymentRolloutRestartAttackDescription))
	exthttp.RegisterHttpHandler("/deployment/attack/rollout-restart/prepare", prepareDeploymentRolloutRestart)
	exthttp.RegisterHttpHandler("/deployment/attack/rollout-restart/start", startDeploymentRolloutRestart)
	exthttp.RegisterHttpHandler("/deployment/attack/rollout-restart/status", deploymentRolloutRestartStatus)
}

func getDeploymentRolloutRestartAttackDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.attack.rollout-restart", deploymentTargetId),
		Label:       "rollout restart deployment",
		Description: "execute a rollout restart for a Kubernetes deployment",
		Version:     "1.0.0",
		Icon:        extutil.Ptr(deploymentIcon),
		TargetType:  extutil.Ptr(deploymentTargetId),
		Category:    extutil.Ptr("state"),
		TimeControl: action_kit_api.Internal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "wait for rollout completion",
				Name:         "wait",
				Type:         action_kit_api.Boolean,
				Advanced:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("false"),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/deployment/attack/rollout-restart/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/deployment/attack/rollout-restart/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method: "POST",
			Path:   "/deployment/attack/rollout-restart/status",
		}),
	}
}

func prepareDeploymentRolloutRestart(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := PrepareDeploymentRolloutRestart(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		utils.WriteActionState(w, *state)
	}
}

func PrepareDeploymentRolloutRestart(body []byte) (*DeploymentRolloutRestartState, *extension_kit.ExtensionError) {
	var request action_kit_api.PrepareActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	wait := false
	if request.Config["wait"] != nil {
		switch v := request.Config["wait"].(type) {
		case bool:
			wait = v
		case string:
			wait = v == "true"
		}
	}

	return extutil.Ptr(DeploymentRolloutRestartState{
		Cluster:    request.Target.Attributes["k8s.cluster-name"][0],
		Namespace:  request.Target.Attributes["k8s.namespace"][0],
		Deployment: request.Target.Attributes["k8s.deployment"][0],
		Wait:       wait,
	}), nil
}

func startDeploymentRolloutRestart(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := StartDeploymentRolloutRestart(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		utils.WriteActionState(w, *state)
	}
}

func StartDeploymentRolloutRestart(body []byte) (*DeploymentRolloutRestartState, *extension_kit.ExtensionError) {
	var request action_kit_api.StartActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var state DeploymentRolloutRestartState
	err = utils.DecodeActionState(request.State, &state)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse attack state", err))
	}

	log.Info().Msgf("Starting deployment rollout restart attack for %+v", state)

	cmd := exec.Command("kubectl",
		"rollout",
		"restart",
		"--namespace",
		state.Namespace,
		fmt.Sprintf("deployment/%s", state.Deployment))
	cmdOut, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to execute rollout restart: %s", cmdOut), cmdErr))
	}

	return &state, nil
}

func deploymentRolloutRestartStatus(w http.ResponseWriter, _ *http.Request, body []byte) {
	result, err := RolloutRestartStatus(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		exthttp.WriteBody(w, result)
	}
}

func RolloutRestartStatus(body []byte) (*action_kit_api.StatusResult, *extension_kit.ExtensionError) {
	var request action_kit_api.ActionStatusRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var state DeploymentRolloutRestartState
	err = utils.DecodeActionState(request.State, &state)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse attack state", err))
	}

	if !state.Wait {
		return extutil.Ptr(action_kit_api.StatusResult{
			Completed: true,
		}), nil
	}

	cmd := exec.Command("kubectl",
		"rollout",
		"status",
		"--watch=false",
		"--namespace",
		state.Namespace,
		fmt.Sprintf("deployment/%s", state.Deployment))
	cmdOut, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to execute rollout restart status check: %s", cmdOut), cmdErr))
	}

	cmdOutStr := string(cmdOut)
	completed := !strings.Contains(strings.ToLower(cmdOutStr), "waiting")
	return extutil.Ptr(action_kit_api.StatusResult{
		Completed: completed,
	}), nil
}
