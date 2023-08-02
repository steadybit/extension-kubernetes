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
		Icon:        extutil.Ptr(logIcon),
		Category:    extutil.Ptr("kubernetes"),
		TimeControl: action_kit_api.Internal,
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

func (f K8sEventsAction) Status(ctx context.Context, state *K8sEventsState) (*action_kit_api.StatusResult, error) {
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

func (f K8sEventsAction) Stop(ctx context.Context, state *K8sEventsState) (*action_kit_api.StopResult, error) {
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
			Message:   event.Message,
			Type:      extutil.Ptr(LogType),
			Level:     convertToLevel(event.Type),
			Timestamp: extutil.Ptr(event.LastTimestamp.Time),
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
