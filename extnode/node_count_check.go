// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"encoding/json"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcluster"
	"math"
	"net/http"
	"time"
)

func RegisterNodeCountCheckHandlers() {
	exthttp.RegisterHttpHandler("/node-count/check", exthttp.GetterAsHandler(getNodeCountCheckDescription))
	exthttp.RegisterHttpHandler("/node-count/check/prepare", prepareNodeCountCheck)
	exthttp.RegisterHttpHandler("/node-count/check/start", startNodeCountCheck)
	exthttp.RegisterHttpHandler("/node-count/check/status", statusNodeCountCheck)
}

const (
	nodeCountAtLeast     = "nodeCountAtLeast"
	nodeCountDecreasedBy = "nodeCountDecreasedBy"
	nodeCountIncreasedBy = "nodeCountIncreasedBy"
)

func getNodeCountCheckDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          nodeCountCheckActionId,
		Label:       "Node Count",
		Description: "Verify node counts",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(nodeCountCheckIcon),
		Category:    extutil.Ptr("kubernetes"),
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.Internal,
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          extcluster.ClusterTargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.ExactlyOne),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find cluster by name"),
					Query:       "k8s.cluster-name=\"\"",
				},
			}),
		}),
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Timeout",
				Description:  extutil.Ptr("How long should the check wait for the specified node count."),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("10s"),
				Order:        extutil.Ptr(1),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "nodeCount",
				Label:        "Node count",
				Description:  extutil.Ptr("How many nodes are required or should change to let the check pass."),
				Type:         action_kit_api.Integer,
				DefaultValue: extutil.Ptr("1"),
				Order:        extutil.Ptr(2),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "nodeCountCheckMode",
				Label:        "Check type",
				Description:  extutil.Ptr("How should the node count change?"),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr(nodeCountAtLeast),
				Order:        extutil.Ptr(2),
				Required:     extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "actual count >= node count",
						Value: nodeCountAtLeast,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "actual count increases by node count",
						Value: nodeCountIncreasedBy,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "actual count decreases by node count",
						Value: nodeCountDecreasedBy,
					},
				}),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/node-count/check/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/node-count/check/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method:       "POST",
			Path:         "/node-count/check/status",
			CallInterval: extutil.Ptr("1s"),
		}),
	}
}

type NodeCountCheckState struct {
	Timeout            time.Time
	NodeCountCheckMode string
	Cluster            string
	NodeCount          int
	InitialNodeCount   int
}

func prepareNodeCountCheck(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := prepareNodeCountCheckInternal(client.K8S, body)
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

func prepareNodeCountCheckInternal(k8s *client.Client, body []byte) (*NodeCountCheckState, *extension_kit.ExtensionError) {
	var request action_kit_api.PrepareActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	duration := math.Round(request.Config["duration"].(float64))
	timeout := time.Now().Add(time.Millisecond * time.Duration(duration))
	cluster := request.Target.Attributes["k8s.cluster-name"][0]
	nodeCountCheckMode := request.Config["nodeCountCheckMode"].(string)
	nodeCount := int(request.Config["nodeCount"].(float64))
	initialNodeCount := k8s.NodesReadyCount()

	return extutil.Ptr(NodeCountCheckState{
		Timeout:            timeout,
		Cluster:            cluster,
		NodeCountCheckMode: nodeCountCheckMode,
		NodeCount:          nodeCount,
		InitialNodeCount:   initialNodeCount,
	}), nil
}

func startNodeCountCheck(w http.ResponseWriter, _ *http.Request, _ []byte) {
	exthttp.WriteBody(w, action_kit_api.StartActionResponse{})
}

func statusNodeCountCheck(w http.ResponseWriter, _ *http.Request, body []byte) {
	result := statusNodeCountCheckInternal(client.K8S, body)
	exthttp.WriteBody(w, result)
}

func statusNodeCountCheckInternal(k8s *client.Client, body []byte) action_kit_api.StatusResult {
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

	var state NodeCountCheckState
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

	readyCount := k8s.NodesReadyCount()

	var checkError *action_kit_api.ActionKitError
	if state.NodeCountCheckMode == nodeCountAtLeast && readyCount < state.NodeCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has not enough ready nodes.", state.Cluster),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.NodeCountCheckMode == nodeCountIncreasedBy && (state.NodeCount+state.InitialNodeCount) != readyCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has only %d of desired %d nodes ready.", state.Cluster, readyCount, state.NodeCount+state.InitialNodeCount),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.NodeCountCheckMode == nodeCountDecreasedBy && (state.InitialNodeCount-state.NodeCount) != readyCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has %d of desired %d nodes ready.", state.Cluster, readyCount, state.InitialNodeCount-state.NodeCount),
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
