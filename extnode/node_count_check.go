// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
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
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1s"),
		}),
	}
}

func (f NodeCountCheckAction) Prepare(_ context.Context, state *NodeCountCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	prepareNodeCountCheckInternal(client.K8S, state, request)
	return nil, nil
}

func prepareNodeCountCheckInternal(k8s *client.Client, state *NodeCountCheckState, request action_kit_api.PrepareActionRequestBody) {
	var config NodeCountCheckConfig
	err := extconversion.Convert(request.Config, &config)
	if err != nil {
		log.Error().Msgf("failed to convert config: %v", err)
	}
	state.Timeout = time.Now().Add(time.Millisecond * time.Duration(config.Duration))
	state.Cluster = request.Target.Attributes["k8s.cluster-name"][0]
	state.NodeCountCheckMode = config.NodeCountCheckMode
	state.NodeCount = config.NodeCount
	state.InitialNodeCount = k8s.NodesReadyCount()
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
