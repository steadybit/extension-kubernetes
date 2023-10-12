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
	"github.com/steadybit/extension-kubernetes/extcluster"
	"github.com/steadybit/extension-kubernetes/extconfig"
	appsv1 "k8s.io/api/apps/v1"
	"time"
)

type PodCountMetricsAction struct {
}

type PodCountMetricsState struct {
	End         time.Time
	LastMetrics map[string]int32
}

type PodCountMetricsConfig struct {
	Duration int
}

func NewPodCountMetricsAction() action_kit_sdk.Action[PodCountMetricsState] {
	return PodCountMetricsAction{}
}

var _ action_kit_sdk.Action[PodCountMetricsState] = (*PodCountMetricsAction)(nil)
var _ action_kit_sdk.ActionWithStatus[PodCountMetricsState] = (*PodCountMetricsAction)(nil)

func (f PodCountMetricsAction) NewEmptyState() PodCountMetricsState {
	return PodCountMetricsState{}
}

func (f PodCountMetricsAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          PodCountMetricActionId,
		Label:       "Pod Count Metrics",
		Description: "Collects information about pod counts (desired vs. actual count).",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xMC40NDc4IDIuNjU2MjVMNC42NTU0NSA1LjU2MDI2QzQuNTg4NzcgNS41OTM2OSA0LjUyMzY5IDUuNjI5NDEgNC40NjAzMSA1LjY2NzI5TDExLjAyODggOS4zMTY0M0MxMS42MzI4IDkuNjUyMDEgMTIuMzY3MyA5LjY1MjAxIDEyLjk3MTMgOS4zMTY0M0wxOS41MDA5IDUuNjg4OTFDMTkuNDMxNSA1LjY0ODg3IDE5LjM2MDIgNS42MTE0IDE5LjI4NzEgNS41NzY2NkwxMy4wNzk0IDIuNjI4MjFDMTIuMjQ0NyAyLjIzMTc0IDExLjI3MzkgMi4yNDIwOSAxMC40NDc4IDIuNjU2MjVaTTIwLjgxNDMgNy4yNDcxNkwxMy45NDI2IDExLjA2NDdDMTMuNjQxOSAxMS4yMzE4IDEzLjMyNSAxMS4zNTczIDEzIDExLjQ0MTJMMTMgMjEuNDA4MUMxMy4wMjY2IDIxLjM5NjQgMTMuMDUzMSAyMS4zODQzIDEzLjA3OTQgMjEuMzcxOEwxOS4yODcxIDE4LjQyMzNDMjAuMzMzMyAxNy45MjY0IDIxIDE2Ljg3MTcgMjEgMTUuNzEzNFY4LjI4NjUyQzIxIDcuOTI1NDIgMjAuOTM1MiA3LjU3NDM3IDIwLjgxNDMgNy4yNDcxNlpNMTEgMjEuNTU1NFYxMS40NDEyQzEwLjY3NSAxMS4zNTczIDEwLjM1ODIgMTEuMjMxOCAxMC4wNTc1IDExLjA2NDdMMy4xNzIzNSA3LjIzOTY4QzMuMDYgNy41NTY0NiAzIDcuODk0NzIgMyA4LjI0MjA4VjE1Ljc1NzlDMyAxNi44OTMgMy42NDA3IDE3LjkzMSA0LjY1NTQ1IDE4LjQzOTdMMTAuNDQ3OCAyMS4zNDM3QzEwLjYyNiAyMS40MzMxIDEwLjgxMTEgMjEuNTAzNyAxMSAyMS41NTU0WiIgZmlsbD0iIzFEMjYzMiIvPgo8Y2lyY2xlIGN4PSIxMiIgY3k9IjEyIiByPSI2IiBmaWxsPSJ3aGl0ZSIvPgo8cGF0aCBkPSJNMTEuNTA0OSAxMC4xNjI5QzExLjQzNTYgOS45OTkzNCAxMS4yNTkzIDkuOTAyMDYgMTEuMDc1NyA5LjkwMDAzQzEwLjg5MjEgOS44OTgwMSAxMC43MTMzIDkuOTkxMjkgMTAuNjM5MiAxMC4xNTI2TDkuNTcxMzQgMTIuNDc2N0g4LjM2MzY0QzguMTI1NyAxMi40NzY3IDcuOSAxMi42NDAzIDcuOSAxMi44Nzg5QzcuOSAxMy4xMTc2IDguMTI1NyAxMy4yODExIDguMzYzNjQgMTMuMjgxMUg5Ljg4NTg0QzEwLjA2NzQgMTMuMjgxMSAxMC4yNDMxIDEzLjE4OCAxMC4zMTY0IDEzLjAyODVMMTEuMDUyIDExLjQyNzVMMTIuNDk1MSAxNC44MzdDMTIuNTYwNSAxNC45OTE0IDEyLjcyMTkgMTUuMDg3NSAxMi44OTUxIDE1LjA5ODhDMTMuMDY4MiAxNS4xMTAyIDEzLjI0MjUgMTUuMDM2NiAxMy4zMzM5IDE0Ljg5NjFMMTQuMzg1MSAxMy4yODExSDE1LjYzNjRDMTUuODc0MyAxMy4yODExIDE2LjEgMTMuMTE3NiAxNi4xIDEyLjg3ODlDMTYuMSAxMi42NDAzIDE1Ljg3NDMgMTIuNDc2NyAxNS42MzY0IDEyLjQ3NjdIMTQuMTE0MkMxMy45NTI2IDEyLjQ3NjcgMTMuNzk1NSAxMi41NTAxIDEzLjcxMDUgMTIuNjgwNkwxMy4wMTk3IDEzLjc0MTlMMTEuNTA0OSAxMC4xNjI5WiIgZmlsbD0iIzFEMjYzMiIgc3Ryb2tlPSIjMUQyNjMyIiBzdHJva2Utd2lkdGg9IjAuMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIi8+Cjwvc3ZnPgo="),
		Category:    extutil.Ptr("Kubernetes"),
		Kind:        action_kit_api.Other,
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
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("2s"),
		}),
	}
}

func (f PodCountMetricsAction) Prepare(_ context.Context, state *PodCountMetricsState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config PodCountMetricsConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}
	state.End = time.Now().Add(time.Millisecond * time.Duration(config.Duration))
	state.LastMetrics = make(map[string]int32)
	return nil, nil
}

func (f PodCountMetricsAction) Start(_ context.Context, _ *PodCountMetricsState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (f PodCountMetricsAction) Status(_ context.Context, state *PodCountMetricsState) (*action_kit_api.StatusResult, error) {
	return statusPodCountMetricsInternal(client.K8S, state), nil
}

func statusPodCountMetricsInternal(k8s *client.Client, state *PodCountMetricsState) *action_kit_api.StatusResult {
	now := time.Now()

	var metrics []action_kit_api.Metric
	for _, d := range k8s.Deployments() {
		if hasChanges(d, state) {
			for _, m := range toMetrics(d, now) {
				state.LastMetrics[getMetricKey(d, *m.Name)] = int32(m.Value)
				metrics = append(metrics, m)
			}
		}
	}

	return &action_kit_api.StatusResult{
		Completed: now.After(state.End),
		Metrics:   extutil.Ptr(metrics),
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
