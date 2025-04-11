// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpod

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
)

func NewDeletePodAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getDeletePodDescription(),
		OptsProvider: deletePod(),
	}
}

func getDeletePodDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          DeletePodActionId,
		Label:       "Delete Pod",
		Description: "Delete Pods in a Kubernetes cluster",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTExLjM2IDE5Ljk5TDExLjAxIDIwLjE2VjExLjE2QzExLjIgMTEuMTEgMTEuNCAxMS4wNSAxMS41OCAxMC45NkwxOC4wNSA3Ljc0OTk5QzE4LjA5IDcuODk5OTkgMTguMTIgOC4wNDk5OSAxOC4xMiA4LjIwOTk5QzE4LjEyIDguNjQ5OTkgMTguNDggOS4wMDk5OSAxOC45MiA5LjAwOTk5QzE5LjM2IDkuMDA5OTkgMTkuNzIgOC42NDk5OSAxOS43MiA4LjIwOTk5QzE5LjcyIDYuOTY5OTkgMTkgNS44Mjk5OSAxNy44OCA1LjI5OTk5TDExLjYxIDIuMzE5OTlDMTAuNzEgMS44ODk5OSA5LjY4IDEuODk5OTkgOC43OCAyLjM0OTk5TDIuODkgNS4yOTk5OUMxLjc5IDUuODQ5OTkgMS4xMSA2Ljk0OTk5IDEuMTEgOC4xNzk5OVYxNS43OEMxLjExIDE3LjAxIDEuNzkgMTguMTEgMi44OSAxOC42Nkw4Ljc4IDIxLjYxQzkuMjQgMjEuODQgOS43MyAyMS45NSAxMC4yMiAyMS45NUMxMC43MSAyMS45NSAxMS4xNiAyMS44NSAxMS42MSAyMS42NEwxMi4wNSAyMS40M0MxMi40NSAyMS4yNCAxMi42MiAyMC43NiAxMi40MyAyMC4zNkMxMi4yNCAxOS45NiAxMS43NiAxOS43OSAxMS4zNiAxOS45OFYxOS45OVpNOS40OCAyMC4xOEwzLjYgMTcuMjNDMy4wNSAxNi45NSAyLjcxIDE2LjQgMi43MSAxNS43OFY4LjE3OTk5QzIuNzEgOC4wMjk5OSAyLjczIDcuODg5OTkgMi43NyA3Ljc0OTk5TDguOTUgMTAuOTNDOS4xMiAxMS4wMiA5LjMgMTEuMDggOS40OSAxMS4xM1YyMC4xOEg5LjQ4Wk05LjY4IDkuNTA5OTlMMy45NSA2LjU0OTk5TDkuNSAzLjc2OTk5QzkuOTQgMy41NDk5OSAxMC40NyAzLjUzOTk5IDEwLjkyIDMuNzU5OTlMMTYuODMgNi41Njk5OUwxMC44OCA5LjUyOTk5QzEwLjUgOS43MTk5OSAxMC4wNiA5LjcwOTk5IDkuNjkgOS41Mjk5OUw5LjY4IDkuNTA5OTlaTTIwLjE5IDEzLjM2QzE5LjkzIDEzLjEgMTkuNTEgMTMuMSAxOS4yNiAxMy4zNkwxNy43NSAxNC44N0wxNi4yNCAxMy4zNkMxNS45OCAxMy4xIDE1LjU2IDEzLjEgMTUuMzEgMTMuMzZDMTUuMDUgMTMuNjIgMTUuMDUgMTQuMDQgMTUuMzEgMTQuMjlMMTYuODIgMTUuOEwxNS4zMSAxNy4zMUMxNS4wNSAxNy41NyAxNS4wNSAxNy45OSAxNS4zMSAxOC4yNEMxNS41NyAxOC40OSAxNS45OSAxOC41IDE2LjI0IDE4LjI0TDE3Ljc1IDE2LjczTDE5LjI2IDE4LjI0QzE5LjUyIDE4LjUgMTkuOTQgMTguNSAyMC4xOSAxOC4yNEMyMC40NCAxNy45OCAyMC40NSAxNy41NiAyMC4xOSAxNy4zMUwxOC42OCAxNS44TDIwLjE5IDE0LjI5QzIwLjQ1IDE0LjAzIDIwLjQ1IDEzLjYxIDIwLjE5IDEzLjM2Wk0xNy43NSA5Ljg2OTk5QzE0LjQ3IDkuODY5OTkgMTEuODEgMTIuNTMgMTEuODEgMTUuODFDMTEuODEgMTkuMDkgMTQuNDcgMjEuNzUgMTcuNzUgMjEuNzVDMjEuMDMgMjEuNzUgMjMuNjkgMTkuMDkgMjMuNjkgMTUuODFDMjMuNjkgMTIuNTMgMjEuMDMgOS44Njk5OSAxNy43NSA5Ljg2OTk5Wk0xNy43NSAyMC40MkMxNS4yIDIwLjQyIDEzLjEzIDE4LjM1IDEzLjEzIDE1LjhDMTMuMTMgMTMuMjUgMTUuMiAxMS4xOCAxNy43NSAxMS4xOEMyMC4zIDExLjE4IDIyLjM3IDEzLjI1IDIyLjM3IDE1LjhDMjIuMzcgMTguMzUgMjAuMyAyMC40MiAxNy43NSAyMC40MloiIGZpbGw9IiMxRDI2MzIiLz4KPC9zdmc+Cg=="),
		Technology:  extutil.Ptr("Kubernetes"),
		TargetSelection: extutil.Ptr(targetSelectionTemplates),
		TimeControl: action_kit_api.TimeControlInternal,
		Kind:        action_kit_api.Attack,
		Parameters:  []action_kit_api.ActionParameter{},
		Prepare:     action_kit_api.MutatingEndpointReference{},
		Start:       action_kit_api.MutatingEndpointReference{},
		Status:      &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:        &action_kit_api.MutatingEndpointReference{},
	}
}

func deletePod() extcommon.KubectlOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*extcommon.KubectlOpts, error) {
		namespace := request.Target.Attributes["k8s.namespace"][0]
		pod := request.Target.Attributes["k8s.pod.name"][0]

		command := []string{"kubectl",
			"delete",
			"pod",
			"--namespace",
			namespace,
			pod}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: nil,
			LogTargetType:   "pod",
			LogTargetName:   pod,
			LogActionName:   "delete pod",
		}, nil
	}
}
