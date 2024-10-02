// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extevents

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcluster"
	corev1 "k8s.io/api/core/v1"
	"os"
	"strings"
	"time"
)

const LogType = "KUBERNETES_EVENTS"

type K8sEventsAction struct {
}

type K8sEventsState struct {
	LastEventTime *int64 `json:"lastEventTime"`
	TimeoutEnd    *int64 `json:"timeoutEnd"`
}

type K8sEventsConfig struct {
	Duration int
}

func NewK8sEventsAction() action_kit_sdk.Action[K8sEventsState] {
	return K8sEventsAction{}
}

var _ action_kit_sdk.Action[K8sEventsState] = (*K8sEventsAction)(nil)
var _ action_kit_sdk.ActionWithStatus[K8sEventsState] = (*K8sEventsAction)(nil)
var _ action_kit_sdk.ActionWithStop[K8sEventsState] = (*K8sEventsAction)(nil)

func (f K8sEventsAction) NewEmptyState() K8sEventsState {
	return K8sEventsState{}
}

func (f K8sEventsAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          "com.steadybit.extension_kubernetes.kubernetes_logs",
		Label:       "Kubernetes Event Logs",
		Description: "Collect event logs from a Kubernetes",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xOS41MiA1LjQ1QzE5Ljg2IDUuNjQgMjAuMTQgNS45MyAyMC4yNSA2LjNIMjAuMjRMMjEuOTggMTMuNzhDMjIuMDMgMTQuMTcgMjEuOTQgMTQuNTYgMjEuNzEgMTQuODhMMTYuODggMjAuODVDMTYuNjMgMjEuMTkgMTYuMjQgMjEuMzggMTUuODMgMjEuMzNIOC4xNUM3Ljc2IDIxLjMxIDcuMzcgMjEuMTIgNy4xIDIwLjg1TDIuMjcgMTQuODhDMi4wNCAxNC41NiAxLjk1IDE0LjE3IDIuMDIgMTMuNzhMMy43MyA2LjI2QzMuODMgNS44NyA0LjA4IDUuNTcgNC40MiA1LjQxTDExLjQgMi4wNUMxMS41OCAyIDExLjc5IDIgMTEuOTcgMkMxMi4xNSAyIDEyLjM2IDIuMDIgMTIuNTQgMi4xMUwxOS41MiA1LjQ1Wk0xOS4wMiAxNC4yNkMxOS4wNiAxNC4yOCAxOS4xMyAxNC4yOCAxOS4xMyAxNC4yOEwxOS4xOCAxNC4zMUMxOS4zOSAxNC4zMSAxOS41NyAxNC4xNyAxOS42NCAxMy45OUMxOS42MiAxMy43NCAxOS40NCAxMy41NiAxOS4yMyAxMy41MUMxOS4yIDEzLjUxIDE5LjE4IDEzLjUxIDE5LjE4IDEzLjQ5QzE5LjE4IDEzLjQ3IDE5LjE0IDEzLjQ3IDE5LjA5IDEzLjQ3QzE5IDEzLjQ1IDE4LjkxIDEzLjQ1IDE4LjgyIDEzLjQ1QzE4Ljc3IDEzLjQ1IDE4LjY4IDEzLjQzIDE4LjY4IDEzLjQzSDE4LjY2QzE4LjQyIDEzLjQgMTguMTYgMTMuMzYgMTcuOTIgMTMuM0MxOCAxMi45IDE4LjA1IDEyLjQ3IDE4LjA1IDEyLjA0QzE4LjA1IDEwLjggMTcuNjggOS42NSAxNy4wMyA4LjY5QzE3LjE4IDguNTggMTcuMzMgOC40OCAxNy40OSA4LjM4TDE3LjYzIDguMzFDMTcuNjYwNyA4LjI5NzcgMTcuNjg3NyA4LjI4MTYzIDE3LjcxMzIgOC4yNjY0MkMxNy43MjkyIDguMjU2ODkgMTcuNzQ0NiA4LjI0NzcgMTcuNzYgOC4yNEMxNy43NzU0IDguMjMyMyAxNy43OTA4IDguMjIzMTEgMTcuODA2OCA4LjIxMzU4QzE3LjgzMjMgOC4xOTgzNyAxNy44NTkzIDguMTgyMyAxNy44OSA4LjE3QzE3LjkgOC4xNiAxNy45MSA4LjE1IDE3LjkyIDguMTVDMTcuOTMgOC4xNSAxNy45NCA4LjE0IDE3Ljk1IDguMTNWOC4xVjguMDhDMTguMTUgNy45MiAxOC4yIDcuNjUgMTguMDQgNy40NEMxNy45NyA3LjM1IDE3LjgzIDcuMjggMTcuNzIgNy4yOEMxNy42MSA3LjI4IDE3LjQ5IDcuMzIgMTcuNCA3LjM5TDE3LjM4IDcuNDFDMTcuMzYgNy40NCAxNy4zMyA3LjQ2IDE3LjMxIDcuNDZDMTcuMjQgNy41MyAxNy4xOCA3LjYgMTcuMTMgNy42N0MxNy4xMTYzIDcuNjkwNSAxNy4wOTggNy43MDYzMyAxNy4wODE0IDcuNzIwNjlDMTcuMDczNyA3LjcyNzM0IDE3LjA2NjMgNy43MzM2NyAxNy4wNiA3Ljc0QzE3LjA1IDcuNzUgMTcuMDMgNy43NiAxNy4wMyA3Ljc2QzE2LjkyIDcuOSAxNi43NyA4LjAzIDE2LjYyIDguMTVDMTUuNTggNi45MyAxNC4wNiA2LjEzIDEyLjM1IDYuMDVDMTIuMzUgNS44OCAxMi4zNiA1LjcyIDEyLjM5IDUuNTRWNS41MkMxMi4zOSA1LjUxIDEyLjM5MjUgNS40OTc1IDEyLjM5NSA1LjQ4NUMxMi4zOTc1IDUuNDcyNSAxMi40IDUuNDYgMTIuNCA1LjQ1QzEyLjQxIDUuNDMgMTIuNDEgNS40IDEyLjQxIDUuMzhDMTIuNDExNSA1LjM3MjcxIDEyLjQxMjkgNS4zNjU2MiAxMi40MTQzIDUuMzU4NjlDMTIuNDIyNyA1LjMxODEyIDEyLjQzIDUuMjgyNzEgMTIuNDMgNS4yNEMxMi40MyA1LjE5IDEyLjQ1IDUuMSAxMi40NSA1LjFWNC45NkMxMi40NyA0LjczIDEyLjI5IDQuNSAxMi4wNiA0LjQ4QzExLjkyIDQuNDYgMTEuNzggNC41MyAxMS42NyA0LjY0QzExLjU4IDQuNzMgMTEuNTMgNC44NSAxMS41MyA0Ljk2VjUuMDdDMTEuNTMgNS4xMzc1IDExLjU0NjkgNS4yMDUgMTEuNTYzNyA1LjI3MjVDMTEuNTY5NCA1LjI5NSAxMS41NzUgNS4zMTc1IDExLjU4IDUuMzRDMTEuNiA1LjM5IDExLjYgNS40OCAxMS42IDUuNDhWNS41QzExLjYzIDUuNjggMTEuNjQgNS44NiAxMS42NCA2LjA0QzkuOTQgNi4xNSA4LjQzIDYuOTcgNy40MSA4LjJDNy4yNCA4LjA2IDcuMDggNy45MSA2Ljk1IDcuNzZDNi45MzYzMyA3LjczOTUgNi45MTc5OSA3LjcyMzY3IDYuOTAxMzcgNy43MDkzMUM2Ljg5MzY3IDcuNzAyNjYgNi44ODYzMyA3LjY5NjMzIDYuODggNy42OUM2Ljg3IDcuNjggNi44NSA3LjY3IDYuODUgNy42N0M2LjgzNSA3LjY1NSA2LjgyIDcuNjM3NSA2LjgwNSA3LjYyQzYuNzkgNy42MDI1IDYuNzc1IDcuNTg1IDYuNzYgNy41N0M2Ljc0NSA3LjU1NSA2LjczIDcuNTM3NSA2LjcxNSA3LjUyQzYuNyA3LjUwMjUgNi42ODUgNy40ODUgNi42NyA3LjQ3QzYuNjYgNy40NiA2LjY1IDcuNDUgNi42NCA3LjQ1QzYuNjMgNy40NSA2LjYxIDcuNDMgNi42MSA3LjQzTDYuNTkgNy40MUM2LjUgNy4zNSA2LjM4IDcuMyA2LjI3IDcuM0M2LjEzIDcuMyA2LjAyIDcuMzUgNS45NSA3LjQ2QzUuODEgNy42NyA1Ljg2IDcuOTQgNi4wNCA4LjFDNi4wNiA4LjEgNi4wNiA4LjEyIDYuMDYgOC4xMkM2LjA2IDguMTIgNi4xMSA4LjE3IDYuMTMgOC4xN0M2LjE3Njk2IDguMjAzNTQgNi4yMzI5MSA4LjIzMjU4IDYuMjkxODMgOC4yNjMxNkM2LjMyMDc1IDguMjc4MTcgNi4zNTAzNyA4LjI5MzU0IDYuMzggOC4zMUw2LjUyIDguMzhMNi41MjAwMSA4LjM4QzYuNyA4LjQ5IDYuODggOC42IDcuMDQgOC43M0M2LjQxIDkuNjggNi4wNCAxMC44MiA2LjA0IDEyLjA1QzYuMDQgMTIuNDYgNi4wOCAxMi44NiA2LjE2IDEzLjI0QzYuMTYgMTMuMjUgNi4xNCAxMy4yNiA2LjE0IDEzLjI2QzUuODkgMTMuMzMgNS42MyAxMy4zOCA1LjM2IDEzLjRDNS4zMSAxMy40IDUuMjcgMTMuNCA1LjIyIDEzLjQyQzUuMTk1IDEzLjQyIDUuMTcyNSAxMy40MjI1IDUuMTUgMTMuNDI1QzUuMTI3NSAxMy40Mjc1IDUuMTA1IDEzLjQzIDUuMDggMTMuNDNDNS4wMyAxMy40NCA0Ljk5MDAxIDEzLjQ0IDQuOTQwMDEgMTMuNDRINC45NEg0LjkxQzQuOSAxMy40NSA0Ljg4IDEzLjQ1IDQuODUgMTMuNDVINC44M0M0LjgyIDEzLjQ1IDQuODEgMTMuNDYgNC44IDEzLjQ3QzQuNTQgMTMuNTIgNC4zOCAxMy43NSA0LjQzIDE0QzQuNDggMTQuMjEgNC42OCAxNC4zNCA0Ljg5IDE0LjMyQzQuOTMgMTQuMzIgNC45NSAxNC4zMiA1IDE0LjNINS4wMlYxNC4yOEM1LjAyIDE0LjI3MzggNS4wMzE0NiAxNC4yNzUzIDUuMDQ3MjkgMTQuMjc3M0M1LjA1NzA4IDE0LjI3ODUgNS4wNjg1NCAxNC4yOCA1LjA4IDE0LjI4SDUuMTFMNS4xMTAxIDE0LjI4QzUuMTcwMDYgMTQuMjYgNS4yMzAwMyAxNC4yNCA1LjI4IDE0LjIyQzUuMjkxMjkgMTQuMjE2MiA1LjMwMTE3IDE0LjIxMTEgNS4zMTA3IDE0LjIwNjFDNS4zMjY0OCAxNC4xOTc4IDUuMzQxMjkgMTQuMTkgNS4zNiAxNC4xOUM1LjQxIDE0LjE2IDUuNSAxNC4xNCA1LjUgMTQuMTRINS41MkM1Ljc3IDE0LjA0IDYgMTMuOTggNi4yNyAxMy45M0g2LjI5SDYuMzJDNi44IDE1LjM3IDcuODEgMTYuNTcgOS4xMiAxNy4yOUM5LjE0IDE3LjM2IDkuMTQgMTcuNDQgOS4xMiAxNy41QzkuMDIgMTcuNzMgOC44OSAxNy45NSA4Ljc1IDE4LjE2VjE4LjE4QzguNzMgMTguMjIgOC43MSAxOC4yNCA4LjY2IDE4LjI5QzguNjQgMTguMzEgOC42MSAxOC4zNSA4LjU4IDE4LjRDOC41NiAxOC40NCA4LjUzMDAxIDE4LjQ4IDguNTAwMDIgMTguNTJMOC41IDE4LjUyQzguNDkgMTguNTMgOC40OCAxOC41NCA4LjQ4IDE4LjU1QzguNDggMTguNTYgOC40NyAxOC41NyA4LjQ2IDE4LjU4QzguNDYgMTguNTggOC40NiAxOC42IDguNDQgMTguNkM4LjMyIDE4LjgzIDguNDEgMTkuMTEgOC42MiAxOS4yMkM4LjY3IDE5LjI1IDguNzMgMTkuMjcgOC43OCAxOS4yN0M4Ljk2IDE5LjI3IDkuMTIgMTkuMTYgOS4yMSAxOUM5LjIxIDE5IDkuMjEgMTguOTggOS4yMyAxOC45OEM5LjIzIDE4Ljk2IDkuMjYgMTguOTMgOS4yOCAxOC45MUM5LjI4Nzg5IDE4Ljg5MDMgOS4yOTQyMiAxOC44NzIxIDkuMzAwMjMgMTguODU0OUM5LjMwOTQ0IDE4LjgyODQgOS4zMTc4OSAxOC44MDQyIDkuMzMgMTguNzhDOS4zNSAxOC43NCA5LjM4IDE4LjY1IDkuMzggMTguNjVMOS40MyAxOC41MUM5LjQ4OTk1IDE4LjI5NTkgOS41ODY1OSAxOC4wOTY0IDkuNjgyMiAxNy44OTkxQzkuNjk4MjIgMTcuODY2IDkuNzE0MjEgMTcuODMzIDkuNzMgMTcuOEM5Ljc3IDE3LjczIDkuODQgMTcuNjggOS45MSAxNy42Nkg5LjkzVjE3LjY1QzEwLjU4IDE3Ljg5IDExLjI4IDE4LjAyIDEyLjAyIDE4LjAyQzEyLjc2IDE4LjAyIDEzLjQ2IDE3Ljg5IDE0LjExIDE3LjY1QzE0LjE4IDE3LjY3IDE0LjI0IDE3LjcyIDE0LjI4IDE3Ljc4QzE0LjQgMTguMDEgMTQuNTEgMTguMjQgMTQuNTggMTguNDlWMTguNTFMMTQuNjMgMTguNjVDMTQuNjUgMTguNzQgMTQuNjcgMTguODMgMTQuNzIgMTguOUMxNC43MyAxOC45MSAxNC43NCAxOC45MiAxNC43NCAxOC45M0MxNC43NCAxOC45NCAxNC43NSAxOC45NSAxNC43NiAxOC45NkwxNC43NiAxOC45NkMxNC43NiAxOC45NiAxNC43NiAxOC45OCAxNC43OCAxOC45OEMxNC44NyAxOS4xNCAxNS4wMyAxOS4yNSAxNS4yMSAxOS4yNUMxNS4yNiAxOS4yNSAxNS4zIDE5LjI0IDE1LjM1IDE5LjIyTDE1LjM3IDE5LjIxSDE1LjM5QzE1LjQ5IDE5LjE3IDE1LjU4IDE5LjA3IDE1LjYgMTguOTZDMTUuNjMgMTguODUgMTUuNjMgMTguNzMgMTUuNTggMTguNjJDMTUuNTggMTguNiAxNS41NiAxOC42IDE1LjU2IDE4LjZDMTUuNTYgMTguNTggMTUuNTMgMTguNTUgMTUuNTEgMTguNTNDMTUuNDYgMTguNDQgMTUuNDIgMTguMzcgMTUuMzUgMTguM0MxNS4zMyAxOC4yNiAxNS4yNiAxOC4xOSAxNS4yNiAxOC4xOVYxOC4xNEMxNS4xIDE3Ljk0IDE0Ljk4IDE3LjcxIDE0Ljg5IDE3LjQ4QzE0Ljg3IDE3LjQyIDE0Ljg2IDE3LjM0IDE0Ljg5IDE3LjI4QzE2LjIyIDE2LjU1IDE3LjI1IDE1LjMzIDE3LjcyIDEzLjg2SDE3Ljc0QzE3Ljk5IDEzLjkxIDE4LjI0IDEzLjk4IDE4LjQ3IDE0LjA3SDE4LjQ5QzE4LjU0IDE0LjEgMTguNTggMTQuMTIgMTguNjMgMTQuMTJDMTguNjYgMTQuMTMgMTguNyAxNC4xNSAxOC43IDE0LjE1QzE4LjcxMjIgMTQuMTU2MSAxOC43MjM5IDE0LjE2MjIgMTguNzM1NSAxNC4xNjgyQzE4Ljc4MTEgMTQuMTkxOCAxOC44MjQyIDE0LjIxNDEgMTguODggMTQuMjNIMTguOTFDMTguOTIgMTQuMjQgMTguOTcgMTQuMjQgMTguOTcgMTQuMjRMMTkuMDIgMTQuMjZaTTE0LjEyIDEyLjM4SDE1LjY0TDE1LjY1IDEyLjM3QzE1Ljk2IDEyLjM3IDE2LjIxIDEyLjU5IDE2LjIxIDEyLjg3QzE2LjIxIDEzLjE1IDE1Ljk2IDEzLjM3IDE1LjY1IDEzLjM3SDE0LjQ1TDEzLjQzIDE0Ljk0QzEzLjMzIDE1LjEgMTMuMTQgMTUuMTkgMTIuOTQgMTUuMTlIMTIuOUMxMi42OCAxNS4xOCAxMi40OSAxNS4wNSAxMi40MSAxNC44N0wxMS4wNiAxMS42N0wxMC40MiAxMy4wN0MxMC4zNCAxMy4yNiAxMC4xMyAxMy4zOCA5LjkgMTMuMzhIOC4zOEM4LjA3IDEzLjM4IDcuODIgMTMuMTUgNy44MiAxMi44OEM3LjgyIDEyLjYxIDguMDcgMTIuMzggOC4zOCAxMi4zOEg5LjUyTDEwLjU2IDEwLjExQzEwLjY1IDkuOTIgMTAuODUgOS44IDExLjA4IDkuOEMxMS4zMSA5LjggMTEuNTIgOS45MyAxMS42IDEwLjEyTDEzLjA0IDEzLjUzTDEzLjYzIDEyLjYzQzEzLjczIDEyLjQ4IDEzLjkyIDEyLjM4IDE0LjEyIDEyLjM4WiIgZmlsbD0iIzFEMjYzMiIvPgo8L3N2Zz4K"),
		Technology:  extutil.Ptr("Kubernetes"),
		Category:    extutil.Ptr("Kubernetes"), //Can be removed in Q1/24 - support for backward compatibility of old sidebar
		TimeControl: action_kit_api.TimeControlInternal,
		Kind:        action_kit_api.Other,
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
				Label:        "Duration",
				Description:  extutil.Ptr(""),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("60s"),
				Order:        extutil.Ptr(1),
				Required:     extutil.Ptr(true),
			},
		},
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.LogWidget{
				Type:    action_kit_api.ComSteadybitWidgetLog,
				Title:   "Kubernetes Events",
				LogType: LogType,
			},
		}),
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{}),
		Stop:    extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

func (f K8sEventsAction) Prepare(_ context.Context, state *K8sEventsState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config K8sEventsConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}

	var timeoutEnd *int64
	if config.Duration != 0 {
		timeoutEnd = extutil.Ptr(time.Now().Add(time.Duration(int(time.Millisecond) * config.Duration)).Unix())
	}
	state.LastEventTime = extutil.Ptr(time.Now().Unix())
	state.TimeoutEnd = timeoutEnd
	return nil, nil
}

func (f K8sEventsAction) Start(_ context.Context, state *K8sEventsState) (*action_kit_api.StartResult, error) {
	state.LastEventTime = extutil.Ptr(time.Now().Unix())
	return nil, nil
}

func (f K8sEventsAction) Status(_ context.Context, state *K8sEventsState) (*action_kit_api.StatusResult, error) {
	return statusInternal(client.K8S, state), nil
}

func statusInternal(k8s *client.Client, state *K8sEventsState) *action_kit_api.StatusResult {
	if state.TimeoutEnd != nil && time.Now().After(time.Unix(*state.TimeoutEnd, 0)) {
		return extutil.Ptr(action_kit_api.StatusResult{
			Completed: true,
		})
	}

	messages := getMessages(k8s, state)
	return extutil.Ptr(action_kit_api.StatusResult{
		Completed: false,
		Messages:  messages,
	})
}

func getMessages(k8s *client.Client, state *K8sEventsState) *action_kit_api.Messages {
	newLastEventTime := time.Now().Unix()
	events := k8s.Events(time.Unix(*state.LastEventTime, 0))
	state.LastEventTime = extutil.Ptr(newLastEventTime)

	// log events
	for _, event := range *events {
		log.Debug().Msgf("Event: %s", event.Message)
	}

	messages := eventsToMessages(events)
	return messages
}

func (f K8sEventsAction) Stop(_ context.Context, state *K8sEventsState) (*action_kit_api.StopResult, error) {
	return stopInternal(client.K8S, state), nil
}

func stopInternal(k8s *client.Client, state *K8sEventsState) *action_kit_api.StopResult {
	messages := getMessages(k8s, state)
	return extutil.Ptr(action_kit_api.StopResult{
		Messages: messages,
	})
}

func eventsToMessages(events *[]corev1.Event) *action_kit_api.Messages {
	var messages []action_kit_api.Message
	clusterName, cnAvailable := os.LookupEnv("STEADYBIT_EXTENSION_CLUSTER_NAME")
	if !cnAvailable {
		clusterName = "unknown"
	}
	for _, event := range *events {
		messages = append(messages, action_kit_api.Message{
			Message:         event.Message,
			Type:            extutil.Ptr(LogType),
			Level:           convertToLevel(event.Type),
			Timestamp:       extutil.Ptr(event.LastTimestamp.Time),
			TimestampSource: extutil.Ptr(action_kit_api.TimestampSourceExternal),
			Fields: extutil.Ptr(action_kit_api.MessageFields{
				"reason":       event.Reason,
				"cluster-name": clusterName,
				"namespace":    event.Namespace,
				"object":       strings.ToLower(event.InvolvedObject.Kind) + "/" + event.InvolvedObject.Name,
			}),
		})
	}
	return extutil.Ptr(messages)
}

func convertToLevel(eventType string) *action_kit_api.MessageLevel {
	switch eventType {
	case "Error":
		return extutil.Ptr(action_kit_api.Error)
	case "Debug":
		return extutil.Ptr(action_kit_api.Debug)
	case "Normal":
		return extutil.Ptr(action_kit_api.Info)
	case "Warning":
		return extutil.Ptr(action_kit_api.Warn)
	default:
		return extutil.Ptr(action_kit_api.Info)
	}
}
