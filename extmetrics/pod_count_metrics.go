// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extmetrics

import (
	"encoding/json"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	appsv1 "k8s.io/api/apps/v1"
	"math"
	"net/http"
	"time"
)

func RegisterPodCountMetricsHandlers() {
	exthttp.RegisterHttpHandler("/metrics/pod-count", exthttp.GetterAsHandler(getPodCountMetricsDescription))
	exthttp.RegisterHttpHandler("/metrics/pod-count/prepare", preparePodCountMetrics)
	exthttp.RegisterHttpHandler("/metrics/pod-count/start", startPodCountMetrics)
	exthttp.RegisterHttpHandler("/metrics/pod-count/status", statusPodCountMetrics)
}

func getPodCountMetricsDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          podCountMetricActionId,
		Label:       "pod count metrics",
		Description: "collects information about pod counts (desired vs. actual count).",
		Version:     "1.0.0-SNAPSHOT",
		Icon:        extutil.Ptr(podCountMetricIcon),
		Category:    extutil.Ptr("kubernetes"),
		Kind:        action_kit_api.Other,
		TimeControl: action_kit_api.Internal,
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
		//TODO: Do we want to make the existing Widget reusable?
		//Widgets: extutil.Ptr([]action_kit_api.Widget{
		//	action_kit_api.StateOverTimeWidget{
		//		Type:  action_kit_api.ComSteadybitWidgetStateOverTime,
		//		Title: "Datadog Monitor Status",
		//		Identity: action_kit_api.StateOverTimeWidgetIdentityConfig{
		//			From: "datadog.monitor.id",
		//		},
		//		Label: action_kit_api.StateOverTimeWidgetLabelConfig{
		//			From: "datadog.monitor.name",
		//		},
		//		State: action_kit_api.StateOverTimeWidgetStateConfig{
		//			From: "state",
		//		},
		//		Tooltip: action_kit_api.StateOverTimeWidgetTooltipConfig{
		//			From: "tooltip",
		//		},
		//		Url: extutil.Ptr(action_kit_api.StateOverTimeWidgetUrlConfig{
		//			From: extutil.Ptr("url"),
		//		}),
		//		Value: extutil.Ptr(action_kit_api.StateOverTimeWidgetValueConfig{
		//			Hide: extutil.Ptr(true),
		//		}),
		//	},
		//}),
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/metrics/pod-count/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/metrics/pod-count/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method:       "POST",
			Path:         "/metrics/pod-count/status",
			CallInterval: extutil.Ptr("2s"),
		}),
	}
}

type PodCountMetricsState struct {
	End time.Time
}

func preparePodCountMetrics(w http.ResponseWriter, _ *http.Request, body []byte) {
	state, err := preparePodCountMetricsInternal(body)
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

func preparePodCountMetricsInternal(body []byte) (*PodCountMetricsState, *extension_kit.ExtensionError) {
	var request action_kit_api.PrepareActionRequestBody
	err := json.Unmarshal(body, &request)
	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError("Failed to parse request body", err))
	}

	duration := math.Round(request.Config["duration"].(float64))
	end := time.Now().Add(time.Millisecond * time.Duration(duration))

	return extutil.Ptr(PodCountMetricsState{
		End: end,
	}), nil
}

func startPodCountMetrics(w http.ResponseWriter, _ *http.Request, _ []byte) {
	exthttp.WriteBody(w, action_kit_api.StartActionResponse{})
}

func statusPodCountMetrics(w http.ResponseWriter, _ *http.Request, body []byte) {
	result := statusPodCountMetricsInternal(client.K8S, body)
	exthttp.WriteBody(w, result)
}

func statusPodCountMetricsInternal(k8s *client.Client, body []byte) action_kit_api.StatusResult {
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

	now := time.Now()

	var state PodCountMetricsState
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

	var metrics []action_kit_api.Metric
	for _, d := range k8s.Deployments() {
		for _, m := range toMetrics(d, now) {
			metrics = append(metrics, m)
		}
	}

	return action_kit_api.StatusResult{
		Completed: now.After(state.End),
		Metrics:   extutil.Ptr(metrics),
	}
}

func toMetrics(deployment *appsv1.Deployment, now time.Time) []action_kit_api.Metric {
	metrics := make([]action_kit_api.Metric, 4)

	metrics[0] = action_kit_api.Metric{
		Name: extutil.Ptr("replicas_desired_count"),
		Metric: map[string]string{
			"k8s.cluster-name": extconfig.Config.ClusterName,
			"k8s.namespace":    deployment.Namespace,
			"k8s.deployment":   deployment.Name,
		},
		Timestamp: now,
		Value:     float64(*deployment.Spec.Replicas),
	}
	metrics[1] = action_kit_api.Metric{
		Name: extutil.Ptr("replicas_current_count"),
		Metric: map[string]string{
			"k8s.cluster-name": extconfig.Config.ClusterName,
			"k8s.namespace":    deployment.Namespace,
			"k8s.deployment":   deployment.Name,
		},
		Timestamp: now,
		Value:     float64(deployment.Status.Replicas),
	}
	metrics[2] = action_kit_api.Metric{
		Name: extutil.Ptr("replicas_ready_count"),
		Metric: map[string]string{
			"k8s.cluster-name": extconfig.Config.ClusterName,
			"k8s.namespace":    deployment.Namespace,
			"k8s.deployment":   deployment.Name,
		},
		Timestamp: now,
		Value:     float64(deployment.Status.ReadyReplicas),
	}
	metrics[3] = action_kit_api.Metric{
		Name: extutil.Ptr("replicas_available_count"),
		Metric: map[string]string{
			"k8s.cluster-name": extconfig.Config.ClusterName,
			"k8s.namespace":    deployment.Namespace,
			"k8s.deployment":   deployment.Name,
		},
		Timestamp: now,
		Value:     float64(deployment.Status.AvailableReplicas),
	}

	return metrics
}
