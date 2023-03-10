// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpodcount

import (
	"encoding/json"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extdeployment"
	"math"
	"net/http"
	"time"
)

func RegisterPodCountCheckHandlers() {
	exthttp.RegisterHttpHandler("/pod-count/check", exthttp.GetterAsHandler(getPodCountCheckDescription))
	exthttp.RegisterHttpHandler("/pod-count/check/prepare", preparePodCountCheck)
	exthttp.RegisterHttpHandler("/pod-count/check/start", startPodCountCheck)
	exthttp.RegisterHttpHandler("/pod-count/check/status", statusPodCountCheck)
}

const (
	podCountMin1                 = "podCountMin1"
	podCountEqualsDesiredCount   = "podCountEqualsDesiredCount"
	podCountLessThanDesiredCount = "podCountLessThanDesiredCount"
)

func getPodCountCheckDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          podCountCheckActionId,
		Label:       "Pod Count",
		Description: "Verify pod counts",
		Version:     "1.0.0-SNAPSHOT",
		Icon:        extutil.Ptr(podCountCheckIcon),
		Category:    extutil.Ptr("kubernetes"),
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.Internal,
		TargetType:  extutil.Ptr(extdeployment.DeploymentTargetType),
		TargetSelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "default",
				Description: extutil.Ptr("Find deployment by cluster, namespace and deployment"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
			},
		}),
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Timeout",
				Description:  extutil.Ptr("How long should the check wait for the specified pod count."),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("10s"),
				Order:        extutil.Ptr(1),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "podCountCheckMode",
				Label:        "Pod count",
				Description:  extutil.Ptr("How many pods are required to let the check pass."),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr("podCountEqualsDesiredCount"),
				Order:        extutil.Ptr(2),
				Required:     extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "ready count > 0",
						Value: podCountMin1,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "ready count = desired count",
						Value: podCountEqualsDesiredCount,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "ready count < desired count",
						Value: podCountLessThanDesiredCount,
					},
				}),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/pod-count/check/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/pod-count/check/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method:       "POST",
			Path:         "/pod-count/check/status",
			CallInterval: extutil.Ptr("1s"),
		}),
	}
}

type PodCountCheckState struct {
	Timeout           time.Time
	PodCountCheckMode string
	Namespace         string
	Deployment        string
}

func preparePodCountCheck(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := preparePodCountCheckInternal(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		var convertedState action_kit_api.ActionState
		err := extconversion.Convert(state, &convertedState)
		if err != nil {
			exthttp.WriteError(w, extension_kit.ToError("Failed to encode action state", err))
		} else {
			exthttp.WriteBody(w, action_kit_api.PrepareResult{
				State: convertedState,
			})
		}
	}
}

func preparePodCountCheckInternal(body []byte) (*PodCountCheckState, *extension_kit.ExtensionError) {
	var request action_kit_api.PrepareActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	duration := math.Round(request.Config["duration"].(float64))
	timeout := time.Now().Add(time.Millisecond * time.Duration(duration))
	podCountCheckMode := request.Config["podCountCheckMode"].(string)
	namespace := request.Target.Attributes["k8s.namespace"][0]
	deployment := request.Target.Attributes["k8s.deployment"][0]

	return extutil.Ptr(PodCountCheckState{
		Timeout:           timeout,
		PodCountCheckMode: podCountCheckMode,
		Namespace:         namespace,
		Deployment:        deployment,
	}), nil
}

func startPodCountCheck(w http.ResponseWriter, _ *http.Request, _ []byte) {
	exthttp.WriteBody(w, action_kit_api.StartActionResponse{})
}

func statusPodCountCheck(w http.ResponseWriter, _ *http.Request, body []byte) {
	result := statusPodCountCheckInternal(client.K8S, body)
	exthttp.WriteBody(w, result)
}

func statusPodCountCheckInternal(k8s *client.Client, body []byte) action_kit_api.StatusResult {
	var request action_kit_api.ActionStatusRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return action_kit_api.StatusResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to parse request body",
				Detail: extutil.Ptr(err.Error()),
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}

	var state PodCountCheckState
	err = extconversion.Convert(request.State, &state)
	if err != nil {
		return action_kit_api.StatusResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to decode action state",
				Detail: extutil.Ptr(err.Error()),
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}

	now := time.Now()

	deployment := k8s.DeploymentByNamespaceAndName(state.Namespace, state.Deployment)
	if deployment == nil {
		return action_kit_api.StatusResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  fmt.Sprintf("Deployment %s not found", state.Deployment),
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}

	readyCount := deployment.Status.ReadyReplicas
	desiredCount := int32(0)
	if deployment.Spec.Replicas != nil {
		desiredCount = *deployment.Spec.Replicas
	} else if state.PodCountCheckMode == podCountEqualsDesiredCount || state.PodCountCheckMode == podCountLessThanDesiredCount {
		return action_kit_api.StatusResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  fmt.Sprintf("Deployment %s has no desired count.", state.Deployment),
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}

	var checkError *action_kit_api.ActionKitError
	if state.PodCountCheckMode == podCountMin1 && readyCount < 1 {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has no ready pods.", state.Deployment),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.PodCountCheckMode == podCountEqualsDesiredCount && readyCount != desiredCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has only %d of desired %d pods ready.", state.Deployment, readyCount, desiredCount),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.PodCountCheckMode == podCountLessThanDesiredCount && readyCount == desiredCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has all %d desired pods ready.", state.Deployment, desiredCount),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	}

	if now.After(state.Timeout) {
		return action_kit_api.StatusResult{
			Completed: true,
			Error:     checkError,
		}
	} else {
		return action_kit_api.StatusResult{
			Completed: checkError == nil,
		}
	}

}
