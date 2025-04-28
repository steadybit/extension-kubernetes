package extingress

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	networkingv1 "k8s.io/api/networking/v1"
	"os/exec"
	"strconv"
	"strings"
)

type HAProxyBlockTrafficState struct {
	ExecutionId    uuid.UUID
	Namespace      string
	IngressName    string
	PathStatusCode map[string]int
}

func NewBlockTrafficAction() action_kit_sdk.Action[HAProxyBlockTrafficState] {
	return &HAProxyBlockTrafficAction{}
}

type HAProxyBlockTrafficAction struct{}

func (a *HAProxyBlockTrafficAction) NewEmptyState() HAProxyBlockTrafficState {
	return HAProxyBlockTrafficState{}
}

func (a *HAProxyBlockTrafficAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          "com.steadybit.extension_kubernetes.haproxy-block-traffic",
		Label:       "HAProxy Block Traffic",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: "Block traffic by returning a custom HTTP status code for requests matching specific paths.",
		Technology:  extutil.Ptr("Kubernetes"),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9IiMyMjIiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIj4KICA8Y2lyY2xlIGN4PSIxMiIgY3k9IjEyIiByPSIxMCIvPgogIDxsaW5lIHgxPSI3IiB5MT0iNyIgeDI9IjE3IiB5Mj0iMTciLz4KPC9zdmc+"),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: HAProxyIngressTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "ingress",
					Description: extutil.Ptr("Find ingress by cluster, namespace and ingress"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.ingress=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Description:  extutil.Ptr("The duration of the action. The ingress will block traffic for the specified duration."),
				Name:         "duration",
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "pathStatusCode",
				Label:        "Path and Statuscode",
				Description:  extutil.Ptr("Key is the path, value is the status code. Example: /block=503"),
				Type:         action_kit_api.KeyValue,
				DefaultValue: extutil.Ptr("[{\"key\"=\"/\", value=\"503\"}]"),
				Required:     extutil.Ptr(true),
			},
		},
	}
}

func (a *HAProxyBlockTrafficAction) Prepare(_ context.Context, state *HAProxyBlockTrafficState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]
	var err error
	if (request.Config["pathStatusCode"]) != nil {
		pathStatusCode := make(map[string]string)
		pathStatusCode, err = extutil.ToKeyValue(request.Config, "pathStatusCode")
		if err != nil {
			return nil, err
		}
		// check status codes
		state.PathStatusCode = make(map[string]int)
		for path, statusCodeStr := range pathStatusCode {
			var statusCode int
			if statusCode, err = strconv.Atoi(statusCodeStr); err != nil {
				return nil, fmt.Errorf("invalid status code: %s", statusCodeStr)
			}
			//append to map
			state.PathStatusCode[path] = statusCode
		}
	}
	//Check ingress availability
	_, err = client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}
	return nil, nil
}

func (a *HAProxyBlockTrafficAction) Start(_ context.Context, state *HAProxyBlockTrafficState) (*action_kit_api.StartResult, error) {
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}

	existingConfig := ingress.Annotations[AnnotationKey]

	// Build the new configuration dynamically using the state.PathStatusCode map
	var configBuilder strings.Builder
	configBuilder.WriteString(getStartMarker(state.ExecutionId) + "\n")
	for path, statusCode := range state.PathStatusCode {
		configBuilder.WriteString(fmt.Sprintf("http-request return status %d if { path %s }\n", statusCode, path))
	}
	configBuilder.WriteString(getEndMarker(state.ExecutionId) + "\n")

	newConfig := configBuilder.String()

	// Prepend the new configuration
	ingress.Annotations[AnnotationKey] = newConfig + "\n" + existingConfig

	err = updateIngress(state.Namespace, state.IngressName, AnnotationKey, ingress)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func updateIngress(namespace string, ingressName string, annotationKey string, ingress *networkingv1.Ingress) error {
	cmd := exec.Command("kubectl", "annotate", "ingress", fmt.Sprintf("%s", ingressName), fmt.Sprintf("%s=%s", annotationKey, ingress.Annotations[annotationKey]), "--overwrite", fmt.Sprintf("--namespace=%s", namespace), "--overwrite")
	cmdOut, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return extension_kit.ToError(fmt.Sprintf("Failed to update ingress: %s", cmdOut), cmdErr)
	}
	return nil
}

func (a *HAProxyBlockTrafficAction) Stop(_ context.Context, state *HAProxyBlockTrafficState) (*action_kit_api.StopResult, error) {
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}

	existingConfig := ingress.Annotations[AnnotationKey]

	// Remove the configuration block for this execution
	updatedConfig := removeConfigBlock(existingConfig, getStartMarker(state.ExecutionId), getEndMarker(state.ExecutionId))

	ingress.Annotations[AnnotationKey] = updatedConfig
	err = updateIngress(state.Namespace, state.IngressName, AnnotationKey, ingress)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
