// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

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
	"github.com/steadybit/extension-kubernetes/extconfig"
	appsv1 "k8s.io/api/apps/v1"
	"math"
	"net/http"
	"time"
)

func RegisterPodCountMetricsHandlers() {
	exthttp.RegisterHttpHandler("/pod-count/metrics", exthttp.GetterAsHandler(getPodCountMetricsDescription))
	exthttp.RegisterHttpHandler("/pod-count/metrics/prepare", preparePodCountMetrics)
	exthttp.RegisterHttpHandler("/pod-count/metrics/start", startPodCountMetrics)
	exthttp.RegisterHttpHandler("/pod-count/metrics/status", statusPodCountMetrics)
}

func getPodCountMetricsDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          podCountMetricActionId,
		Label:       "Pod Count Metrics",
		Description: "Collects information about pod counts (desired vs. actual count).",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(podCountMetricIcon),
		Category:    extutil.Ptr("kubernetes"),
		Kind:        action_kit_api.Other,
		TimeControl: action_kit_api.Internal,
		TargetType:  extutil.Ptr(extcluster.ClusterTargetType),
		TargetSelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "default",
				Description: extutil.Ptr("Find cluster by name"),
				Query:       "k8s.cluster-name=\"\"",
			},
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
			action_kit_api.PredefinedWidget{
				Type:               action_kit_api.ComSteadybitWidgetPredefined,
				PredefinedWidgetId: "com.steadybit.widget.predefined.DeploymentReadinessWidget",
			},
		}),
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/pod-count/metrics/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/pod-count/metrics/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method:       "POST",
			Path:         "/pod-count/metrics/status",
			CallInterval: extutil.Ptr("2s"),
		}),
	}
}

type PodCountMetricsState struct {
	End         time.Time
	LastMetrics map[string]int32
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
		End:         end,
		LastMetrics: make(map[string]int32),
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
		if hasChanges(d, &state) {
			for _, m := range toMetrics(d, now) {
				state.LastMetrics[getMetricKey(d, *m.Name)] = int32(m.Value)
				metrics = append(metrics, m)
			}
		}
	}

	var convertedState action_kit_api.ActionState
	err = extconversion.Convert(state, &convertedState)
	if err != nil {
		return action_kit_api.StatusResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to encode action state",
				Detail: extutil.Ptr(err.Error()),
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}

	return action_kit_api.StatusResult{
		Completed: now.After(state.End),
		Metrics:   extutil.Ptr(metrics),
		State:     &convertedState,
	}
}

func hasChanges(deployment *appsv1.Deployment, state *PodCountMetricsState) bool {
	currentDesiredReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		currentDesiredReplicas = *deployment.Spec.Replicas
	}

	return hasChange(deployment, state, "replicas_current_count", deployment.Status.Replicas) ||
		hasChange(deployment, state, "replicas_desired_count", currentDesiredReplicas) ||
		hasChange(deployment, state, "replicas_ready_count", deployment.Status.ReadyReplicas) ||
		hasChange(deployment, state, "replicas_available_count", deployment.Status.AvailableReplicas)
}

func hasChange(deployment *appsv1.Deployment, state *PodCountMetricsState, metric string, currentValue int32) bool {
	key := getMetricKey(deployment, metric)
	oldValue, oldValuePresent := state.LastMetrics[key]
	return !oldValuePresent || oldValue != currentValue
}

func getMetricKey(deployment *appsv1.Deployment, metric string) string {
	return fmt.Sprintf("%s-%s/%s", metric, deployment.Namespace, deployment.Name)
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
