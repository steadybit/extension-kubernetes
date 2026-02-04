// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"context"
	"fmt"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcluster"
)

const (
	nodeCountAtLeast     = "nodeCountAtLeast"
	nodeCountDecreasedBy = "nodeCountDecreasedBy"
	nodeCountIncreasedBy = "nodeCountIncreasedBy"
)

var referenceTime = time.Now()

type NodeCountCheckAction struct {
}

type NodeCountCheckState struct {
	EndOffset          time.Duration
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
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik02LjQ5IDkuNjNMMi41OSA4LjE2QzIuMjMgOC4wMyAyIDcuNjkgMiA3LjMxQzIgNi45MyAyLjIzIDYuNTkgMi41OSA2LjQ2TDExLjYgMy4wNkMxMS44IDIuOTggMTIuMDMgMi45OCAxMi4yMyAzLjA2TDIxLjI0IDYuNDZDMjEuNiA2LjU5IDIxLjgzIDYuOTMgMjEuODMgNy4zMUMyMS44MyA3LjY5IDIxLjU5IDguMDMgMjEuMjQgOC4xNkwxNy40OSA5LjU4QzE2LjU1IDcuNDcgMTQuNDcgNiAxMiA2QzkuNTMgNiA3LjQxIDcuNDkgNi40OSA5LjYzWk0xNCAxMC4wMUwxMS4xNyAxMi45OEwxMC4wMSAxMS43NUM5Ljc0IDExLjQ3IDkuMyAxMS40NyA5LjAyIDExLjczQzguNzQgMTIgOC43NCAxMi40NCA5IDEyLjcyTDEwLjY2IDE0LjQ3QzEwLjc5IDE0LjYxIDEwLjk3IDE0LjY4IDExLjE2IDE0LjY4QzExLjM1IDE0LjY4IDExLjUzIDE0LjYgMTEuNjYgMTQuNDdMMTUgMTAuOTdDMTUuMjcgMTAuNjkgMTUuMjYgMTAuMjUgMTQuOTggOS45OEMxNC43IDkuNzEgMTQuMjYgOS43MiAxMy45OSAxMEwxNCAxMC4wMVpNMy4yMTk5OCAxMS4yM0MyLjc0OTk4IDExLjA1IDIuMjI5OTggMTEuMjkgMi4wNTk5OCAxMS43NkMxLjg4OTk4IDEyLjIzIDIuMTE5OTggMTIuNzUgMi41ODk5OCAxMi45M0w2LjUxOTk4IDE0LjQxQzYuMjI5OTggMTMuNzUgNi4wNTk5OCAxMy4wNCA2LjAxOTk4IDEyLjI4TDMuMjE5OTggMTEuMjJWMTEuMjNaTTIwLjYgMTYuMDFMMTEuOTEgMTkuMjlMMy4yMTk5OCAxNi4wMUMyLjc0OTk4IDE1LjgzIDIuMjI5OTggMTYuMDcgMi4wNTk5OCAxNi41NEMxLjg4OTk4IDE3LjAxIDIuMTE5OTggMTcuNTMgMi41ODk5OCAxNy43MUwxMS42IDIxLjExQzExLjggMjEuMTkgMTIuMDMgMjEuMTkgMTIuMjMgMjEuMTFMMjEuMjQgMTcuNzFDMjEuNzEgMTcuNTMgMjEuOTQgMTcuMDEgMjEuNzcgMTYuNTRDMjEuNiAxNi4wNyAyMS4wOCAxNS44MyAyMC42MSAxNi4wMUgyMC42Wk0xNy45OCAxMi4yMkwyMC42IDExLjIzQzIxLjA3IDExLjA1IDIxLjU5IDExLjI5IDIxLjc2IDExLjc2QzIxLjkzIDEyLjIzIDIxLjcgMTIuNzUgMjEuMjMgMTIuOTNMMTcuNTIgMTQuMzNDMTcuOCAxMy42OCAxNy45NSAxMi45NyAxNy45OCAxMi4yMloiIGZpbGw9IiMxRDI2MzIiLz4KPC9zdmc+Cg=="),
		Technology:  extutil.Ptr("Kubernetes"),
		Category:    extutil.Ptr("Kubernetes"), //Can be removed in Q1/24 - support for backward compatibility of old sidebar
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          extcluster.ClusterTargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.QuantityRestrictionExactlyOne),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "cluster name",
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
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("10s"),
				Order:        extutil.Ptr(1),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "nodeCount",
				Label:        "Node count",
				Description:  extutil.Ptr("How many nodes are required or should change to let the check pass."),
				Type:         action_kit_api.ActionParameterTypeInteger,
				DefaultValue: extutil.Ptr("1"),
				Order:        extutil.Ptr(2),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "nodeCountCheckMode",
				Label:        "Check type",
				Description:  extutil.Ptr("How should the node count change?"),
				Type:         action_kit_api.ActionParameterTypeString,
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
	duration := time.Duration(int(time.Millisecond) * config.Duration)
	state.EndOffset = time.Since(referenceTime) + duration
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

	if time.Since(referenceTime) > state.EndOffset {
		return &action_kit_api.StatusResult{
			Completed: true,
			Error:     checkError,
		}
	}

	return &action_kit_api.StatusResult{
		Completed: checkError == nil,
	}
}
