// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

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
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
)

func NewSetImageAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getSetImageDescription(),
		OptsProvider: setImage(),
	}
}

type SetImageConfig struct {
	Image         string `json:"image"`
	ContainerName string `json:"container_name"`
}

func getSetImageDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          SetImageActionId,
		Label:       "Set Image",
		Description: "Set Image for Kubernetes Deployment",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0yMC4xMSAxNS41SDE4LjM2QzE3Ljk1IDE1LjUgMTcuNjEgMTUuMTYgMTcuNjEgMTQuNzVDMTcuNjEgMTQuMzQgMTcuOTUgMTQgMTguMzYgMTRIMjAuMTFDMjAuOCAxNCAyMS4zNiAxMy40NCAyMS4zNiAxMi43NVY0Ljc1QzIxLjM2IDQuMDYgMjAuOCAzLjUgMjAuMTEgMy41SDkuMTA5OTlDOC40MTk5OSAzLjUgNy44NTk5OSA0LjA2IDcuODU5OTkgNC43NVY1LjkzQzcuODU5OTkgNi4zNCA3LjUxOTk5IDYuNjggNy4xMDk5OSA2LjY4QzYuNjk5OTkgNi42OCA2LjM1OTk5IDYuMzQgNi4zNTk5OSA1LjkzVjQuNzVDNi4zNTk5OSAzLjIzIDcuNTg5OTkgMiA5LjEwOTk5IDJIMjAuMTFDMjEuNjMgMiAyMi44NiAzLjIzIDIyLjg2IDQuNzVWMTIuNzVDMjIuODYgMTQuMjcgMjEuNjMgMTUuNSAyMC4xMSAxNS41Wk0xOS4zNSA1SDE3LjE3QzE2Ljg5IDUgMTYuNjUgNS4yMyAxNi42NSA1LjUyQzE2LjY1IDUuODEgMTYuODggNi4wNCAxNy4xNyA2LjA0SDE4LjFMMTYuNiA3LjUxVjcuNTNMMTQuMDUgMTAuMDVWOS4xMUMxNC4wNSA4LjgzIDEzLjgyIDguNTkgMTMuNTMgOC41OUMxMy4yNCA4LjU5IDEzLjAxIDguODIgMTMuMDEgOS4xMVYxMS4yOUMxMy4wMSAxMS41NyAxMy4yNCAxMS44MSAxMy41MyAxMS44MUgxNS43MUMxNS45OSAxMS44MSAxNi4yMyAxMS41OCAxNi4yMyAxMS4yOUMxNi4yMyAxMSAxNiAxMC43NyAxNS43MSAxMC43N0gxNC43OEwxNy4zMyA4LjI1VjguMjNMMTguODQgNi43NVY3LjY5QzE4Ljg0IDcuOTcgMTkuMDcgOC4yMSAxOS4zNiA4LjIxQzE5LjY1IDguMjEgMTkuODggNy45OCAxOS44OCA3LjY5VjUuNTJDMTkuODggNS4yNCAxOS42NSA1IDE5LjM2IDVIMTkuMzVaTTkuMDM4MTUgMTAuMjE1MkwxMi44OTY1IDEyLjE2NTJWMTIuMTczNkMxMy41NzQ5IDEyLjUyMTIgMTQgMTMuMjE2NCAxNCAxMy45Nzk0VjE5LjAzMjNDMTQgMTkuNzk1MyAxMy41NzQ5IDIwLjQ5OSAxMi44OTY1IDIwLjgzODFMOS4wMzgxNSAyMi43ODhDOC40ODIyOSAyMy4wNjc4IDcuODM2NTEgMjMuMDY3OCA3LjI4MDY1IDIyLjgwNUwzLjE0NDQxIDIwLjgyMTFDMi40NDE0MiAyMC40OTA1IDIgMTkuNzc4MyAyIDE4Ljk5ODRWMTQuMDA0OUMyIDEzLjIyNDkgMi40NDE0MiAxMi41MTI4IDMuMTQ0NDEgMTIuMTgyMUw3LjI4MDY1IDEwLjE5ODNDNy44MzY1MSA5LjkyNjk5IDguNDkwNDYgOS45MzU0NyA5LjAzODE1IDEwLjIxNTJaTTEyLjI5OTcgMTkuNjI1N0MxMi41Mjg2IDE5LjUwNyAxMi42Njc2IDE5LjI3ODEgMTIuNjY3NiAxOS4wMjM4VjE4Ljk5ODRWMTMuOTQ1NUMxMi42Njc2IDEzLjY5MTIgMTIuNTI4NiAxMy40NTM4IDEyLjI5OTcgMTMuMzQzNkw4LjQ0MTQyIDExLjM5MzdDOC4yNTM0MSAxMS4zMDg5IDguMDQwODcgMTEuMzA4OSA3Ljg1Mjg2IDExLjM5MzdMMy43MTY2MiAxMy4zNzc1QzMuNDc5NTYgMTMuNDg3NyAzLjMzMjQyIDEzLjcyNTEgMy4zMzI0MiAxMy45ODc5VjE4Ljk4MTRDMy4zMzI0MiAxOS4yNDQyIDMuNDg3NzQgMTkuNDgxNiAzLjcxNjYyIDE5LjU5MThMNy44NTI4NiAyMS41NzU3QzguMDQwODcgMjEuNjY4OSA4LjI2MTU4IDIxLjY2ODkgOC40NDE0MiAyMS41NzU3TDEyLjI5OTcgMTkuNjI1N1pNOC41ODAzOCAxMy4yNDE5TDEwLjcyMjEgMTQuMjUwN0MxMS4wOTgxIDE0LjQyODggMTEuMzM1MSAxNC43ODQ4IDExLjMzNTEgMTUuMTgzM1YxNy44MDNDMTEuMzM1MSAxOC4xOTMgMTEuMDk4MSAxOC41NTc1IDEwLjcyMjEgMTguNzM1Nkw4LjU4MDM4IDE5Ljc0NDRDOC4yNzc5MyAxOS44ODg2IDcuOTE4MjYgMTkuODg4NiA3LjYwNzYzIDE5Ljc1MjlMNS4zMTA2MyAxOC43MjcxQzQuOTE4MjYgMTguNTU3NSA0LjY3MzAyIDE4LjE5MyA0LjY3MzAyIDE3Ljc4NlYxNS4yMDAzQzQuNjczMDIgMTQuODAxOCA0LjkyNjQzIDE0LjQyODggNS4zMTA2MyAxNC4yNTkyTDcuNjA3NjMgMTMuMjMzNEM3LjkxODI2IDEzLjA5NzcgOC4yNzc5MyAxMy4wOTc3IDguNTgwMzggMTMuMjQxOVoiIGZpbGw9IiMxRDI2MzIiLz4KPC9zdmc+Cg=="),
		Technology:  extutil.Ptr("Kubernetes"),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          DeploymentTargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.ExactlyOne),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "Deployment",
					Description: extutil.Ptr("Find deployment by cluster, namespace, and name"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label: "Duration",
				Description: extutil.Ptr(
					"The duration of the action. The image will be reverted back to the original value after the action.",
				),
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("180s"),
				Required:     extutil.Ptr(true),
			},
			{
				Label: "Container Name",
				Description: extutil.Ptr(
					"The name of the container to set the image for.",
				),
				Name:     "container_name",
				Type:     action_kit_api.ActionParameterTypeString,
				Required: extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ParameterOptionsFromTargetAttribute{
						Attribute: "k8s.container.name",
					},
				}),
			},
			{
				Label:       "Image",
				Name:        "image",
				Description: extutil.Ptr("The new image."),
				Type:        action_kit_api.ActionParameterTypeString,
				Required:    extutil.Ptr(true),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:    &action_kit_api.MutatingEndpointReference{},
	}
}

func setImage() extcommon.KubectlOptsProvider {
	return func(
		ctx context.Context,
		request action_kit_api.PrepareActionRequestBody,
	) (*extcommon.KubectlOpts, error) {
		namespace := request.Target.Attributes["k8s.namespace"][0]
		deployment := request.Target.Attributes["k8s.deployment"][0]

		var config SetImageConfig
		if err := extconversion.Convert(request.Config, &config); err != nil {
			return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
		}

		container := config.ContainerName
		if container == "" {
			return nil, extension_kit.ToError(
				"Container name is required.", nil,
			)
		}

		deploymentDefinition := client.K8S.DeploymentByNamespaceAndName(namespace, deployment)

		if deploymentDefinition == nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find deployment %s/%s.", namespace, deployment), nil)
		}

		if deploymentDefinition.Spec.Template.Spec.Containers == nil {
			return nil, extension_kit.ToError(fmt.Sprintf(
				"Failed to find containers spec for deployment %s/%s.",
				namespace,
				deployment,
			), nil)
		}

		var oldContainerImage string

		for _, c := range deploymentDefinition.Spec.Template.Spec.Containers {
			if c.Name == container {
				oldContainerImage = c.Image
				break
			}
		}

		if oldContainerImage == "" {
			return nil, extension_kit.ToError(fmt.Sprintf(
				"Failed to find container %s in deployment %s/%s.",
				container,
				namespace,
				deployment,
			), nil)
		}

		command := []string{"kubectl",
			"set",
			"image",
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("deployment/%s", deployment),
			fmt.Sprintf("%s=%s", container, config.Image),
		}

		rollbackCommand := []string{"kubectl",
			"set",
			"image",
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("deployment/%s", deployment),
			fmt.Sprintf("%s=%s", container, oldContainerImage),
		}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: &rollbackCommand,
			LogTargetType:   "container",
			LogTargetName: fmt.Sprintf(
				"%s/%s/%s",
				namespace,
				deployment,
				container,
			),
			LogActionName: "set image",
		}, nil
	}
}
