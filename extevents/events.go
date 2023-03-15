// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extevents

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/utils"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"os"
	"strings"
	"time"
)

type K8sEventsState struct {
	LastEventTime *int64 `json:"lastEventTime"`
	TimeoutEnd    *int64 `json:"timeoutEnd"`
}

const LogType = "KUBERNETES_EVENTS"

func RegisterK8sEventsHandlers() {
	exthttp.RegisterHttpHandler("/events", exthttp.GetterAsHandler(getK8sEventsDescription))
	exthttp.RegisterHttpHandler("/events/prepare", prepareK8sEvents)
	exthttp.RegisterHttpHandler("/events/start", startK8sEvents)
	exthttp.RegisterHttpHandler("/events/status", statusK8sEvents)
	exthttp.RegisterHttpHandler("/events/stop", stopK8sEvents)
}

func getK8sEventsDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          "com.github.steadybit.extension_kubernetes.kubernetes_logs",
		Label:       "Kubernetes Event Logs",
		Description: "Collect event logs from a Kubernetes",
		Version:     "1.0.0-SNAPSHOT",
		Icon:        extutil.Ptr(logIcon),
		Category:    extutil.Ptr("kubernetes"),
		TimeControl: action_kit_api.Internal,
		Kind:        action_kit_api.Other,
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
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/events/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/events/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method: "POST",
			Path:   "/events/status",
		}),
	}
}

func prepareK8sEvents(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := PrepareK8sEvents(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		utils.WriteActionState(w, *state)
	}
}

func PrepareK8sEvents(body []byte) (*K8sEventsState, *extension_kit.ExtensionError) {
	var request action_kit_api.PrepareActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var timeoutEnd *int64
	if request.Config["duration"] != nil {
		timeoutEnd = extutil.Ptr(time.Now().Add(time.Duration(float64(time.Millisecond) * request.Config["duration"].(float64))).Unix())
	}

	return extutil.Ptr(K8sEventsState{
		LastEventTime: extutil.Ptr(time.Now().Unix()),
		TimeoutEnd:    timeoutEnd,
	}), nil
}

func startK8sEvents(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := StartK8sLogs(body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		utils.WriteActionState(w, *state)
	}
}

func StartK8sLogs(body []byte) (*K8sEventsState, *extension_kit.ExtensionError) {
	var request action_kit_api.StartActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var state K8sEventsState
	err = utils.DecodeActionState(request.State, &state)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse state", err))
	}

	state.LastEventTime = extutil.Ptr(time.Now().Unix())
	return &state, nil
}

func statusK8sEvents(w http.ResponseWriter, _ *http.Request, body []byte) {
	result, timeout, err := K8sLogsStatus(client.K8S, body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		if timeout {
			log.Info().Msgf("Timeout")
		}
		exthttp.WriteBody(w, result)
	}
}

func K8sLogsStatus(k8s *client.Client, body []byte) (*action_kit_api.StatusResult, bool, *extension_kit.ExtensionError) {
	var request action_kit_api.ActionStatusRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, false, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var state K8sEventsState
	err = utils.DecodeActionState(request.State, &state)
	if err != nil {
		return nil, false, extutil.Ptr(extension_kit.ToError("Failed to parse state", err))
	}

	if state.TimeoutEnd != nil && time.Now().After(time.Unix(*state.TimeoutEnd, 0)) {
		return extutil.Ptr(action_kit_api.StatusResult{
			Completed: true,
			Messages: extutil.Ptr(action_kit_api.Messages{
				action_kit_api.Message{
					Level:   extutil.Ptr(action_kit_api.Error),
					Message: fmt.Sprintf("Timed out reached"),
				},
			}),
		}), true, nil
	}

	messages := getMessages(k8s, state)
	return extutil.Ptr(action_kit_api.StatusResult{
		Completed: false,
		Messages:  messages,
	}), false, nil
}

func getMessages(k8s *client.Client, state K8sEventsState) *action_kit_api.Messages {
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

func stopK8sEvents(w http.ResponseWriter, _ *http.Request, body []byte) {
	result, err := K8sLogsStop(client.K8S, body)
	if err != nil {
		exthttp.WriteError(w, *err)
	} else {
		exthttp.WriteBody(w, result)
	}
}
func K8sLogsStop(k8s *client.Client, body []byte) (*action_kit_api.StopResult, *extension_kit.ExtensionError) {
	var request action_kit_api.ActionStatusRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	var state K8sEventsState
	err = utils.DecodeActionState(request.State, &state)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse state", err))
	}

	messages := getMessages(k8s, state)

	return extutil.Ptr(action_kit_api.StopResult{
		Messages: messages,
	}), nil
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
