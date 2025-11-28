/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package ai

import (
	"context"
	"fmt"
	"github.com/steadybit/extension-kubernetes/v2/extdeployment"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

var (
	_ action_kit_sdk.Action[AnalysisState] = (*reliabilityCheckDeploymentAction)(nil)
)

func NewReliabilityCheckDeploymentAction(converse ConverseWrapper) action_kit_sdk.Action[AnalysisState] {
	return &reliabilityCheckDeploymentAction{converse: converse}
}

func (a *reliabilityCheckDeploymentAction) NewEmptyState() AnalysisState {
	return AnalysisState{}
}

func (a *reliabilityCheckDeploymentAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          "com.steadybit.extension_kubernetes.ai.issues.check-deployment",
		Label:       "Check Deployment",
		Description: "Uses an AI model to analyze a manifest for reliability issues.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(targetIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: extdeployment.DeploymentTargetType,
		}),
		Technology: extutil.Ptr("AI"),
		Category:   extutil.Ptr("Reliability"),

		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.MarkdownWidget{
				Type:        action_kit_api.ComSteadybitWidgetMarkdown,
				Title:       "Reliability Issues",
				MessageType: "ReliabilityIssues",
				Append:      true,
			},
		}),
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("3s"),
		}),
	}
}

// Prepare is called before the action is started.
// It validates and copies the config into the state.
func (a *reliabilityCheckDeploymentAction) Prepare(ctx context.Context, state *AnalysisState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.Kind = "deployment"

	err := prepare(ctx, state, request)
	if err != nil {
		return nil, err
	}

	return &action_kit_api.PrepareResult{
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("ReliabilityIssues"),
				Message: "# AI Analysis",
			},
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("ReliabilityIssues"),
				Message: "---",
			},
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("ReliabilityIssues"),
				Message: fmt.Sprintf("## Preparation\nManifest retrieved for %s %s in namespace %s for cluster %s.  \n\n", state.Kind, state.Name, state.Namespace, state.ClusterName),
			},
		}),
	}, nil
}

func (a *reliabilityCheckDeploymentAction) Start(ctx context.Context, state *AnalysisState) (*action_kit_api.StartResult, error) {
	jobID := fmt.Sprintf("job-%d", time.Now().UnixNano())
	state.JobID = jobID

	reliabilityJobs.mu.Lock()
	reliabilityJobs.m[jobID] = &ReliabilityJob{Done: false}
	reliabilityJobs.mu.Unlock()

	go func() {
		result, err := a.converse.FindReliabilityIssues(
			state.Technology,
			state.Kind,
			state.Manifest,
			nil,
		)

		reliabilityJobs.mu.Lock()
		defer reliabilityJobs.mu.Unlock()
		job := reliabilityJobs.m[jobID]
		job.Done = true
		job.Result = result
		job.Err = err
		job.Timestamp = time.Now()
		storeSingleReliabilityIssues(state.Key, job.Result, job.Timestamp)
	}()

	return &action_kit_api.StartResult{
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("ReliabilityIssues"),
				Message: "###  Analyzing ⟡˙⋆",
			},
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Type:    extutil.Ptr("ReliabilityIssues"),
				Message: "Waiting for completion",
			},
		}),
	}, nil
}

func (a *reliabilityCheckDeploymentAction) Status(ctx context.Context, state *AnalysisState) (*action_kit_api.StatusResult, error) {
	return status(state, "ReliabilityIssues", ReliabilityIssuesToMarkdown)
}
