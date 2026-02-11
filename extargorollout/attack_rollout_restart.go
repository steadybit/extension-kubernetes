// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extargorollout

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ArgoRolloutRestartAction struct {
	k8s *client.Client
}

type ArgoRolloutRestartState struct {
	Namespace   string `json:"namespace"`
	ArgoRollout string `json:"argo-rollout"`
}

func NewArgoRolloutRestartAction(k8s *client.Client) action_kit_sdk.Action[ArgoRolloutRestartState] {
	return &ArgoRolloutRestartAction{k8s: k8s}
}

var _ action_kit_sdk.Action[ArgoRolloutRestartState] = (*ArgoRolloutRestartAction)(nil)

func (a *ArgoRolloutRestartAction) NewEmptyState() ArgoRolloutRestartState {
	return ArgoRolloutRestartState{}
}

func (a *ArgoRolloutRestartAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ArgoRolloutRestartActionId,
		Label:       "Restart Argo Rollout",
		Description: "Trigger a restart of an Argo Rollout by patching spec.restartAt",
		Icon:        extutil.Ptr(ArgoRolloutIcon),
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Technology:  extutil.Ptr("Kubernetes"),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: ArgoRolloutTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label: "by cluster, namespace and rollout",
					Query: "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.argo-rollout=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlInstantaneous,
		Kind:        action_kit_api.Attack,
		Parameters:  []action_kit_api.ActionParameter{},
		Prepare:     action_kit_api.MutatingEndpointReference{},
		Start:       action_kit_api.MutatingEndpointReference{},
	}
}

func (a *ArgoRolloutRestartAction) Prepare(_ context.Context, state *ArgoRolloutRestartState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	ns := request.Target.Attributes["k8s.namespace"]
	ro := request.Target.Attributes["k8s.argo-rollout"]

	if len(ns) == 0 || len(ro) == 0 {
		var missingAttributes []string
		if len(ns) == 0 {
			missingAttributes = append(missingAttributes, "k8s.namespace")
		}
		if len(ro) == 0 {
			missingAttributes = append(missingAttributes, "k8s.argo-rollout")
		}
		return nil, extension_kit.ToError(
			fmt.Sprintf("Missing required target attribute(s): %v", missingAttributes),
			nil,
		)
	}

	state.Namespace = ns[0]
	state.ArgoRollout = ro[0]
	return nil, nil
}

func (a *ArgoRolloutRestartAction) Start(ctx context.Context, state *ArgoRolloutRestartState) (*action_kit_api.StartResult, error) {
	log.Info().Msgf("Restarting Argo Rollout %s/%s", state.Namespace, state.ArgoRollout)

	// Patch spec.restartAt with current time
	restartAt := time.Now().UTC().Format(time.RFC3339)
	patchData := map[string]interface{}{
		"spec": map[string]interface{}{
			"restartAt": restartAt,
		},
	}
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to marshal patch data: %v", err), err)
	}

	_, err = a.k8s.DynamicClient().Resource(client.ArgoRolloutGVR).Namespace(state.Namespace).Patch(
		ctx,
		state.ArgoRollout,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to restart Argo Rollout: %v", err), err)
	}

	log.Info().Msgf("Successfully patched Argo Rollout %s/%s", state.Namespace, state.ArgoRollout)
	return &action_kit_api.StartResult{
		Messages: extutil.Ptr([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("Restart triggered for Argo Rollout %s/%s", state.Namespace, state.ArgoRollout),
			},
		}),
	}, nil
}
