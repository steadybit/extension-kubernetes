// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

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
}

type PodCountCheckState struct {
	Timeout           time.Time
	PodCountCheckMode string
	Namespace         string
	Deployment        string
	InitialCount      int
}
type PodCountCheckConfig struct {
	Duration          int
	PodCountCheckMode string
}

func NewPodCountCheckAction() action_kit_sdk.Action[PodCountCheckState] {
	return PodCountCheckAction{}
}

var _ action_kit_sdk.Action[PodCountCheckState] = (*PodCountCheckAction)(nil)
var _ action_kit_sdk.ActionWithStatus[PodCountCheckState] = (*PodCountCheckAction)(nil)

func (f PodCountCheckAction) NewEmptyState() PodCountCheckState {
	return PodCountCheckState{}
}

func (f PodCountCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          PodCountCheckActionId,
		Label:       "Pod Count",
		Description: "Verify pod counts",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xMC40NDc4IDIuNjU2MjVMNC42NTU0NSA1LjU2MDI2QzQuNTg4NzcgNS41OTM2OSA0LjUyMzY5IDUuNjI5NDEgNC40NjAzMSA1LjY2NzI5TDExLjAyODggOS4zMTY0M0MxMS42MzI4IDkuNjUyMDEgMTIuMzY3MyA5LjY1MjAxIDEyLjk3MTMgOS4zMTY0M0wxOS41MDA5IDUuNjg4OTFDMTkuNDMxNSA1LjY0ODg3IDE5LjM2MDIgNS42MTE0IDE5LjI4NzEgNS41NzY2NkwxMy4wNzk0IDIuNjI4MjFDMTIuMjQ0NyAyLjIzMTc0IDExLjI3MzkgMi4yNDIwOSAxMC40NDc4IDIuNjU2MjVaTTIwLjgxNDMgNy4yNDcxNkwxMy45NDI2IDExLjA2NDdDMTMuNjQxOSAxMS4yMzE4IDEzLjMyNSAxMS4zNTczIDEzIDExLjQ0MTJMMTMgMjEuNDA4MUMxMy4wMjY2IDIxLjM5NjQgMTMuMDUzMSAyMS4zODQzIDEzLjA3OTQgMjEuMzcxOEwxOS4yODcxIDE4LjQyMzNDMjAuMzMzMyAxNy45MjY0IDIxIDE2Ljg3MTcgMjEgMTUuNzEzNFY4LjI4NjUyQzIxIDcuOTI1NDIgMjAuOTM1MiA3LjU3NDM3IDIwLjgxNDMgNy4yNDcxNlpNMTEgMjEuNTU1NFYxMS40NDEyQzEwLjY3NSAxMS4zNTczIDEwLjM1ODIgMTEuMjMxOCAxMC4wNTc1IDExLjA2NDdMMy4xNzIzNSA3LjIzOTY4QzMuMDYgNy41NTY0NiAzIDcuODk0NzIgMyA4LjI0MjA4VjE1Ljc1NzlDMyAxNi44OTMgMy42NDA3IDE3LjkzMSA0LjY1NTQ1IDE4LjQzOTdMMTAuNDQ3OCAyMS4zNDM3QzEwLjYyNiAyMS40MzMxIDEwLjgxMTEgMjEuNTAzNyAxMSAyMS41NTU0WiIgZmlsbD0iIzFEMjYzMiIvPgo8Y2lyY2xlIGN4PSIxMiIgY3k9IjEyIiByPSI2IiBmaWxsPSJ3aGl0ZSIvPgo8cGF0aCBkPSJNMTQuMDY4MyAxMC4wODMzTDExLjE2MTYgMTMuMTM1N0w5LjkzMjAxIDExLjgzMzZMOS45MzIwMSAxMS44MzM2TDkuOTMxMTYgMTEuODMyN0M5LjcwMDcyIDExLjU5NDYgOS4zMjA4NyAxMS41ODg0IDkuMDgyNzUgMTEuODE4OEM4Ljg0NDc0IDEyLjA0OTIgOC44Mzg0IDEyLjQyODcgOS4wNjg0OSAxMi42NjY5QzkuMDY4NjEgMTIuNjY3IDkuMDY4NzIgMTIuNjY3MSA5LjA2ODg0IDEyLjY2NzNMMTAuNzI5NSAxNC40MTY2TDEwLjcyOTUgMTQuNDE2NkwxMC43MzAxIDE0LjQxNzNDMTAuODQzMiAxNC41MzQxIDEwLjk5ODcgMTQuNiAxMS4xNjEzIDE0LjZDMTEuMzIzOCAxNC42IDExLjQ3OTQgMTQuNTM0MSAxMS41OTI1IDE0LjQxNzNMMTEuNTkyOSAxNC40MTY3TDE0LjkzMTIgMTAuOTE3M0MxNC45MzEyIDEwLjkxNzIgMTQuOTMxMyAxMC45MTcxIDE0LjkzMTQgMTAuOTE3QzE1LjE2MTYgMTAuNjc4OSAxNS4xNTUzIDEwLjI5OTIgMTQuOTE3MyAxMC4wNjg4QzE0LjY3OTEgOS44Mzg0IDE0LjI5OTMgOS44NDQ2MiAxNC4wNjg4IDEwLjA4MjdMMTQuMDY4MyAxMC4wODMzWiIgZmlsbD0iIzFEMjYzMiIgc3Ryb2tlPSIjMUQyNjMyIiBzdHJva2Utd2lkdGg9IjAuMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIi8+Cjwvc3ZnPgo="),
		Category:    extutil.Ptr("Kubernetes"),
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          DeploymentTargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.All),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find deployment by cluster, namespace and deployment"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
				},
			}),
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
	return preparePodCountCheckInternal(client.K8S, state, request)
}

func preparePodCountCheckInternal(k8s *client.Client, state *PodCountCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config PodCountCheckConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}

	namespace := request.Target.Attributes["k8s.namespace"][0]
	deployment := request.Target.Attributes["k8s.deployment"][0]
	d := k8s.DeploymentByNamespaceAndName(namespace, deployment)
	if d == nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to find deployment %s/%s.", namespace, deployment), nil)
	}

	state.Timeout = time.Now().Add(time.Millisecond * time.Duration(config.Duration))
	state.PodCountCheckMode = config.PodCountCheckMode
	state.Namespace = namespace
	state.Deployment = deployment
	state.InitialCount = int(d.Status.ReadyReplicas)
	return nil, nil
}

func (f PodCountCheckAction) Start(_ context.Context, _ *PodCountCheckState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (f PodCountCheckAction) Status(_ context.Context, state *PodCountCheckState) (*action_kit_api.StatusResult, error) {
	return statusPodCountCheckInternal(client.K8S, state), nil
}

func statusPodCountCheckInternal(k8s *client.Client, state *PodCountCheckState) *action_kit_api.StatusResult {
	now := time.Now()

	deployment := k8s.DeploymentByNamespaceAndName(state.Namespace, state.Deployment)
	if deployment == nil {
		return &action_kit_api.StatusResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  fmt.Sprintf("Deployment %s not found", state.Deployment),
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}

	readyCount := int(deployment.Status.ReadyReplicas)
	desiredCount := 0
	if deployment.Spec.Replicas != nil {
		desiredCount = int(*deployment.Spec.Replicas)
	} else if state.PodCountCheckMode == podCountEqualsDesiredCount || state.PodCountCheckMode == podCountLessThanDesiredCount {
		return &action_kit_api.StatusResult{
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
	} else if state.PodCountCheckMode == podCountIncreased && readyCount <= state.InitialCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s's pod count didn't increase. Initial count: %d, current count: %d.", state.Deployment, state.InitialCount, readyCount),
			Status: extutil.Ptr(action_kit_api.Failed),
		})
	} else if state.PodCountCheckMode == podCountDecreased && readyCount >= state.InitialCount {
		checkError = extutil.Ptr(action_kit_api.ActionKitError{
			Title:  fmt.Sprintf("%s's pod count didn't decrease. Initial count: %d, current count: %d.", state.Deployment, state.InitialCount, readyCount),
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
