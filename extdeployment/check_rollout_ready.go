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
	"time"
)

type CheckDeploymentRolloutStatusState struct {
	Cluster    string `json:"cluster"`
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	TimeoutEnd *int64 `json:"timeoutEnd"`
}

func RegisterDeploymentRolloutStatusCheckHandlers() {
	exthttp.RegisterHttpHandler("/deployment/check/rollout-status", exthttp.GetterAsHandler(getDeploymentRolloutStatusDescription))
	exthttp.RegisterHttpHandler("/deployment/check/rollout-status/prepare", prepareDeploymentRolloutStatus)
	exthttp.RegisterHttpHandler("/deployment/check/rollout-status/start", startDeploymentRolloutStatus)
	exthttp.RegisterHttpHandler("/deployment/check/rollout-status/status", deploymentRolloutStatusStatus)
}

func getDeploymentRolloutStatusDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.check.rollout-status", deploymentTargetType),
		Label:       "deployment rollout status",
		Description: "Check the rollout status of the deployment. The check succeeds when no rollout is pending, i.e., `kubectl rollout status` exits with status code `0`.",
		Version:     "1.0.0-SNAPSHOT",
		Icon:        extutil.Ptr(deploymentIcon),
		TargetType:  extutil.Ptr(deploymentTargetType),
		Category:    extutil.Ptr("state"),
		TimeControl: action_kit_api.Internal,
		Kind:        action_kit_api.Check,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Timeout",
				Description:  extutil.Ptr("Maximum time to wait for the rollout to be rolled out completely."),
				Name:         "duration",
				Type:         action_kit_api.Duration,
				Advanced:     extutil.Ptr(false),
				DefaultValue: extutil.Ptr("10m"),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/deployment/check/rollout-status/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/deployment/check/rollout-status/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method: "POST",
			Path:   "/deployment/check/rollout-status/status",
		}),
	}
}

func prepareDeploymentRolloutStatus(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := PrepareDeploymentRolloutStatus(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		utils.WriteActionState(w, *state)
	}
}

func PrepareDeploymentRolloutStatus(body []byte) (*CheckDeploymentRolloutStatusState, *extension_kit.ExtensionError) {
	var request action_kit_api.PrepareActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var timeoutEnd *int64
	if request.Config["duration"] != nil {
		timeoutEnd = extutil.Ptr(time.Now().Add(time.Duration(float64(time.Millisecond) * request.Config["duration"].(float64))).Unix())
	}

	return extutil.Ptr(CheckDeploymentRolloutStatusState{
		Cluster:    request.Target.Attributes["k8s.cluster-name"][0],
		Namespace:  request.Target.Attributes["k8s.namespace"][0],
		Deployment: request.Target.Attributes["k8s.deployment"][0],
		TimeoutEnd: timeoutEnd,
	}), nil
}

func startDeploymentRolloutStatus(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := StartDeploymentRolloutStatus(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		utils.WriteActionState(w, *state)
	}
}

func StartDeploymentRolloutStatus(body []byte) (*CheckDeploymentRolloutStatusState, *extension_kit.ExtensionError) {
	var request action_kit_api.StartActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var state CheckDeploymentRolloutStatusState
	err = utils.DecodeActionState(request.State, &state)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse action state", err))
	}

	return &state, nil
}

func deploymentRolloutStatusStatus(w http.ResponseWriter, _ *http.Request, body []byte) {
	result, timeout, err := RolloutStatusStatus(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		statusCode := 200
		if timeout {
			statusCode = 500
		}
		w.WriteHeader(statusCode)
		encodeErr := json.NewEncoder(w).Encode(result)
		if encodeErr != nil {
			log.Err(encodeErr).Msgf("Failed to encode response body")
		}
	}
}

func RolloutStatusStatus(body []byte) (*action_kit_api.StatusResult, bool, *extension_kit.ExtensionError) {
	var request action_kit_api.ActionStatusRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, false, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var state CheckDeploymentRolloutStatusState
	err = utils.DecodeActionState(request.State, &state)
	if err != nil {
		return nil, false, extutil.Ptr(extension_kit.ToError("Failed to parse check state", err))
	}

	if state.TimeoutEnd != nil && time.Now().After(time.Unix(*state.TimeoutEnd, 0)) {
		return extutil.Ptr(action_kit_api.StatusResult{
			Completed: false,
			Messages: extutil.Ptr(action_kit_api.Messages{
				action_kit_api.Message{
					Level:   extutil.Ptr(action_kit_api.Error),
					Message: fmt.Sprintf("Timed out waiting for deployment '%s' in namespace '%s' to complete rollout", state.Deployment, state.Namespace),
				},
			}),
		}), true, nil
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
		return nil, false, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to execute rollout status check: %s", cmdOut), cmdErr))
	}

	cmdOutStr := string(cmdOut)
	completed := !strings.Contains(strings.ToLower(cmdOutStr), "waiting")
	return extutil.Ptr(action_kit_api.StatusResult{
		Completed: completed,
	}), false, nil
}
