// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extcommon

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
	"time"
)

const (
	podCountMin1                 = "podCountMin1"
	podCountEqualsDesiredCount   = "podCountEqualsDesiredCount"
	podCountLessThanDesiredCount = "podCountLessThanDesiredCount"
	podCountDecreased            = "podCountDecreased"
	podCountIncreased            = "podCountIncreased"
)

type PodCountCheckAction struct {
	Client                       *client.Client
	ActionId                     string
	TargetType                   string
	TargetTypeLabel              string
	SelectionTemplates           *action_kit_api.TargetSelectionTemplates
	GetTarget                    func(request action_kit_api.PrepareActionRequestBody) string
	GetDesiredAndCurrentPodCount func(k8s *client.Client, namespace string, target string) (*int32, int32, error)
}

type PodCountCheckState struct {
	Timeout           time.Time
	PodCountCheckMode string
	Namespace         string
	Target            string
	InitialCount      int
}
type PodCountCheckConfig struct {
	Duration          int
	PodCountCheckMode string
}

var _ action_kit_sdk.Action[PodCountCheckState] = (*PodCountCheckAction)(nil)
var _ action_kit_sdk.ActionWithStatus[PodCountCheckState] = (*PodCountCheckAction)(nil)

func (f PodCountCheckAction) NewEmptyState() PodCountCheckState {
	return PodCountCheckState{}
}

func (f PodCountCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          f.ActionId,
		Label:       f.TargetTypeLabel + " Pod Count",
		Description: "Verify " + f.TargetTypeLabel + " pod counts",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xMiA1LjY2MjY4QzEzLjU3IDUuNjYyNjggMTUgNi4yNjI2OCAxNi4wNyA3LjI1MjY4TDE5LjUgNS4zNTI2OEwxOS41IDUuMzUyNjZDMTkuNDMgNS4zMTI2NyAxOS4zNiA1LjI3MjY4IDE5LjI5IDUuMjQyNjhMMTMuMDggMi4yOTI2OEMxMi4yNSAxLjg5MjY4IDExLjI3IDEuOTAyNjggMTAuNDUgMi4zMjI2OEw0LjY2MDAyIDUuMjIyNjhDNC42MDkwMyA1LjI0NDU0IDQuNTYzMzUgNS4yNzE2OSA0LjUxNTI0IDUuMzAwMjlMNC41MTUyMiA1LjMwMDNDNC40OTcyOSA1LjMxMDk2IDQuNDc5MDMgNS4zMjE4MiA0LjQ2MDAyIDUuMzMyNjhMNy45MzAwMiA3LjI2MjY4QzkuMDAwMDIgNi4yNzI2OCAxMC40MyA1LjY3MjY4IDEyIDUuNjcyNjhWNS42NjI2OFpNNi42OSA4Ljg2MjY4QzYuMjUwNzIgOS42OTEzMiA2LjAwMDgyIDEwLjY0OTUgNiAxMS42NTc3TDYgMTEuNjUyN1YxMS42NjI3TDYgMTEuNjU3N0M2LjAwMjQyIDE0LjYzNTQgOC4xNjE1OSAxNy4wOTMgMTEgMTcuNTcyN1YyMS4yMTI3QzEwLjgxIDIxLjE2MjcgMTAuNjMgMjEuMDkyNyAxMC40NSAyMS4wMDI3TDQuNjYgMTguMTAyN0MzLjY0IDE3LjU5MjcgMyAxNi41NjI3IDMgMTUuNDIyN1Y3LjkwMjY4QzMgNy41NjI2OCAzLjA2IDcuMjIyNjggMy4xNyA2LjkwMjY4TDYuNjkgOC44NjI2OFpNMjAuODA1IDYuOTE1NDZMMjAuODEgNi45MTI2OEwyMC44IDYuOTAyNjhMMjAuODA1IDYuOTE1NDZaTTIwLjgwNSA2LjkxNTQ2TDE3LjMgOC44NjI2OEMxNy43NCA5LjcwMjY4IDE3Ljk5IDEwLjY1MjcgMTcuOTkgMTEuNjYyN0MxNy45OSAxNC42MzI3IDE1LjgzIDE3LjEwMjcgMTIuOTkgMTcuNTgyN1YyMS4wNzI3QzEyLjk5IDIxLjA3MjcgMTMuMDQgMjEuMDUyNyAxMy4wNyAyMS4wMzI3TDE5LjI4IDE4LjA4MjdDMjAuMzMgMTcuNTgyNyAyMC45OSAxNi41MzI3IDIwLjk5IDE1LjM3MjdWNy45NDI2OEMyMC45OSA3LjU4NzMzIDIwLjkzMTUgNy4yNDE3MSAyMC44MDUgNi45MTU0NlpNMTQgOS42ODI2OEMxNC4yNyA5LjQwMjY4IDE0LjcxIDkuMzkyNjggMTQuOTkgOS42NjI2OEwxNC45OCA5LjY1MjY4QzE1LjI2IDkuOTIyNjggMTUuMjcgMTAuMzYyNyAxNSAxMC42NDI3TDExLjY2IDE0LjE0MjdDMTEuNTMgMTQuMjcyNyAxMS4zNSAxNC4zNTI3IDExLjE2IDE0LjM1MjdDMTAuOTcgMTQuMzUyNyAxMC43OSAxNC4yODI3IDEwLjY2IDE0LjE0MjdMOSAxMi4zOTI3QzguNzQgMTIuMTEyNyA4Ljc0IDExLjY3MjcgOS4wMiAxMS40MDI3QzkuMyAxMS4xNDI3IDkuNzQgMTEuMTQyNyAxMC4wMSAxMS40MjI3TDExLjE3IDEyLjY1MjdMMTQgOS42ODI2OFoiIGZpbGw9IiMxRDI2MzIiLz4KPC9zdmc+Cg=="),
		Technology:  extutil.Ptr("Kubernetes"),
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          f.TargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.All),
			SelectionTemplates:  f.SelectionTemplates,
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
					action_kit_api.ExplicitParameterOption{
						Label: "actual count increases",
						Value: podCountIncreased,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "actual count decreases",
						Value: podCountDecreased,
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

func (f PodCountCheckAction) Prepare(_ context.Context, state *PodCountCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config PodCountCheckConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}

	namespace := request.Target.Attributes["k8s.namespace"][0]
	target := f.GetTarget(request)
	desired, current, err := f.GetDesiredAndCurrentPodCount(f.Client, namespace, target)
	if err != nil {
		return nil, err
	}
	if desired == nil && (config.PodCountCheckMode == podCountEqualsDesiredCount || config.PodCountCheckMode == podCountLessThanDesiredCount) {
		return nil, extension_kit.ToError(fmt.Sprintf("%s/%s has no desired count", namespace, target), nil)
	}

	state.Timeout = time.Now().Add(time.Millisecond * time.Duration(config.Duration))
	state.PodCountCheckMode = config.PodCountCheckMode
	state.Namespace = namespace
	state.Target = target
	state.InitialCount = int(current)
	return nil, nil
}

func (f PodCountCheckAction) Start(_ context.Context, _ *PodCountCheckState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (f PodCountCheckAction) Status(_ context.Context, state *PodCountCheckState) (*action_kit_api.StatusResult, error) {
	now := time.Now()

	desired, current, err := f.GetDesiredAndCurrentPodCount(f.Client, state.Namespace, state.Target)
	if err != nil {
		return nil, err
	}

	readyCount := int(current)
	desiredCount := int(*desired)

	var checkError *action_kit_api.ActionKitError
	if state.PodCountCheckMode == podCountMin1 && readyCount < 1 {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has no ready pods.", state.Target),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.PodCountCheckMode == podCountEqualsDesiredCount && readyCount != desiredCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has only %d of desired %d pods ready.", state.Target, readyCount, desiredCount),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.PodCountCheckMode == podCountLessThanDesiredCount && readyCount == desiredCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s has all %d desired pods ready.", state.Target, desiredCount),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.PodCountCheckMode == podCountIncreased && readyCount <= state.InitialCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s's pod count didn't increase. Initial count: %d, current count: %d.", state.Target, state.InitialCount, readyCount),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.PodCountCheckMode == podCountDecreased && readyCount >= state.InitialCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s's pod count didn't decrease. Initial count: %d, current count: %d.", state.Target, state.InitialCount, readyCount),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	}

	if now.After(state.Timeout) {
		return &action_kit_api.StatusResult{
			Completed: true,
			Error:     checkError,
		}, nil
	} else {
		return &action_kit_api.StatusResult{
			Completed: checkError == nil,
		}, nil
	}

}
