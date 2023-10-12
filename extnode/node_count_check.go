// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcluster"
	"time"
)

const (
	nodeCountAtLeast     = "nodeCountAtLeast"
	nodeCountDecreasedBy = "nodeCountDecreasedBy"
	nodeCountIncreasedBy = "nodeCountIncreasedBy"
)

type NodeCountCheckAction struct {
}

type NodeCountCheckState struct {
	Timeout            time.Time
	NodeCountCheckMode string
	Cluster            string
	NodeCount          int
	InitialNodeCount   int
}

type NodeCountCheckConfig struct {
	Duration           int
	NodeCountCheckMode string
	NodeCount          int
}

func NewNodeCountCheckAction() action_kit_sdk.Action[NodeCountCheckState] {
	return NodeCountCheckAction{}
}

var _ action_kit_sdk.Action[NodeCountCheckState] = (*NodeCountCheckAction)(nil)
var _ action_kit_sdk.ActionWithStatus[NodeCountCheckState] = (*NodeCountCheckAction)(nil)

func (f NodeCountCheckAction) NewEmptyState() NodeCountCheckState {
	return NodeCountCheckState{}
}

func (f NodeCountCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          NodeCountCheckActionId,
		Label:       "Node Count",
		Description: "Verify node counts",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTExLjU5NDggMy4wNTc3M0MxMS43OTg2IDIuOTgwNzYgMTIuMDIzMiAyLjk4MDc2IDEyLjIyNyAzLjA1NzczTDIxLjIzNjUgNi40NTk5QzIxLjU4ODQgNi41OTI4IDIxLjgyMTYgNi45MzE5MSAyMS44MjE2IDcuMzEwNzdDMjEuODIxNSA3LjY4OTYzIDIxLjU4ODMgOC4wMjg2OSAyMS4yMzYzIDguMTYxNTFMMTIuMjI2OCAxMS41NjE2QzEyLjAyMzEgMTEuNjM4NSAxMS43OTg3IDExLjYzODUgMTEuNTk0OSAxMS41NjE2TDIuNTg1NDYgOC4xNjE1MUMyLjIzMzUgOC4wMjg2OSAyLjAwMDI2IDcuNjg5NjMgMi4wMDAyMiA3LjMxMDc3QzIuMDAwMTggNi45MzE5MSAyLjIzMzM2IDYuNTkyOCAyLjU4NTI5IDYuNDU5OUwxMS41OTQ4IDMuMDU3NzNaIiBmaWxsPSIjMUQyNjMyIi8+CjxwYXRoIGQ9Ik0yLjA1NzUxIDE2LjU0MDZDMi4yMzIwOSAxNi4wNzA4IDIuNzUxNDYgMTUuODMyNSAzLjIxNzU0IDE2LjAwODZMMTEuOTEwOSAxOS4yOTE0TDIwLjYwNDMgMTYuMDA4NkMyMS4wNzA0IDE1LjgzMjUgMjEuNTg5NyAxNi4wNzA4IDIxLjc2NDMgMTYuNTQwNkMyMS45Mzg5IDE3LjAxMDUgMjEuNzAyNiAxNy41MzQxIDIxLjIzNjUgMTcuNzEwMUwxMi4yMjcgMjEuMTEyM0MxMi4wMjMyIDIxLjE4OTIgMTEuNzk4NiAyMS4xODkyIDExLjU5NDggMjEuMTEyM0wyLjU4NTMxIDE3LjcxMDFDMi4xMTkyMyAxNy41MzQxIDEuODgyOTIgMTcuMDEwNSAyLjA1NzUxIDE2LjU0MDZaIiBmaWxsPSIjMUQyNjMyIi8+CjxwYXRoIGQ9Ik0zLjIxNzU0IDExLjIzNDJDMi43NTE0NiAxMS4wNTgyIDIuMjMyMDkgMTEuMjk2NCAyLjA1NzUxIDExLjc2NjNDMS44ODI5MiAxMi4yMzYyIDIuMTE5MjMgMTIuNzU5OCAyLjU4NTMxIDEyLjkzNThMMTEuNTk0OCAxNi4zMzc5QzExLjc5ODYgMTYuNDE0OSAxMi4wMjMyIDE2LjQxNDkgMTIuMjI3IDE2LjMzNzlMMjEuMjM2NSAxMi45MzU4QzIxLjcwMjYgMTIuNzU5OCAyMS45Mzg5IDEyLjIzNjIgMjEuNzY0MyAxMS43NjYzQzIxLjU4OTcgMTEuMjk2NCAyMS4wNzA0IDExLjA1ODIgMjAuNjA0MyAxMS4yMzQyTDExLjkxMDkgMTQuNTE3TDMuMjE3NTQgMTEuMjM0MloiIGZpbGw9IiMxRDI2MzIiLz4KPGNpcmNsZSBjeD0iMTIiIGN5PSIxMiIgcj0iNiIgZmlsbD0id2hpdGUiLz4KPHBhdGggZD0iTTE0LjA2ODMgMTAuMDgzM0wxMS4xNjE2IDEzLjEzNTdMOS45MzIwMSAxMS44MzM2TDkuOTMyMDEgMTEuODMzNkw5LjkzMTE2IDExLjgzMjdDOS43MDA3MiAxMS41OTQ2IDkuMzIwODcgMTEuNTg4NCA5LjA4Mjc1IDExLjgxODhDOC44NDQ3NCAxMi4wNDkyIDguODM4NCAxMi40Mjg3IDkuMDY4NDkgMTIuNjY2OUM5LjA2ODYxIDEyLjY2NyA5LjA2ODcyIDEyLjY2NzEgOS4wNjg4NCAxMi42NjczTDEwLjcyOTUgMTQuNDE2NkwxMC43Mjk1IDE0LjQxNjZMMTAuNzMwMSAxNC40MTczQzEwLjg0MzIgMTQuNTM0MSAxMC45OTg3IDE0LjYgMTEuMTYxMyAxNC42QzExLjMyMzggMTQuNiAxMS40Nzk0IDE0LjUzNDEgMTEuNTkyNSAxNC40MTczTDExLjU5MjkgMTQuNDE2N0wxNC45MzEyIDEwLjkxNzNDMTQuOTMxMiAxMC45MTcyIDE0LjkzMTMgMTAuOTE3MSAxNC45MzE0IDEwLjkxN0MxNS4xNjE2IDEwLjY3ODkgMTUuMTU1MyAxMC4yOTkyIDE0LjkxNzMgMTAuMDY4OEMxNC42NzkxIDkuODM4NCAxNC4yOTkzIDkuODQ0NjIgMTQuMDY4OCAxMC4wODI3TDE0LjA2ODMgMTAuMDgzM1oiIGZpbGw9IiMxRDI2MzIiIHN0cm9rZT0iIzFEMjYzMiIgc3Ryb2tlLXdpZHRoPSIwLjIiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLWxpbmVqb2luPSJyb3VuZCIvPgo8L3N2Zz4K"),
		Category:    extutil.Ptr("Kubernetes"),
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
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
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1s"),
		}),
	}
}

func (f NodeCountCheckAction) Prepare(_ context.Context, state *NodeCountCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	return prepareNodeCountCheckInternal(client.K8S, state, request)
}

func prepareNodeCountCheckInternal(k8s *client.Client, state *NodeCountCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config NodeCountCheckConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}
	state.Timeout = time.Now().Add(time.Millisecond * time.Duration(config.Duration))
	state.Cluster = request.Target.Attributes["k8s.cluster-name"][0]
	state.NodeCountCheckMode = config.NodeCountCheckMode
	state.NodeCount = config.NodeCount
	state.InitialNodeCount = k8s.NodesReadyCount()
	return nil, nil
}

func (f NodeCountCheckAction) Start(_ context.Context, _ *NodeCountCheckState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (f NodeCountCheckAction) Status(_ context.Context, state *NodeCountCheckState) (*action_kit_api.StatusResult, error) {
	return statusNodeCountCheckInternal(client.K8S, state), nil
}

func statusNodeCountCheckInternal(k8s *client.Client, state *NodeCountCheckState) *action_kit_api.StatusResult {
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
		return &action_kit_api.StatusResult{
			Completed: true,
			Error:     checkError,
		}
	} else {
		return &action_kit_api.StatusResult{
			Completed: checkError == nil,
		}
	}

}
