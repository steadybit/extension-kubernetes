package ephemeral_storage_request

import (
	"embed"
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/extadvice/advice_common"
	"github.com/steadybit/extension-kubernetes/extdaemonset"
	"github.com/steadybit/extension-kubernetes/extdeployment"
	"github.com/steadybit/extension-kubernetes/extstatefulset"
)

const EphemeralStorageRequestID = "com.steadybit.extension_kubernetes.advice.k8s-ephemeral-storage-request"

//go:embed *
var EphemeralStorageRequestContent embed.FS

func GetAdviceDescriptionEphemeralStorageRequest() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        EphemeralStorageRequestID,
		Label:                     "Requesting Reasonable Ephemeral Storage Resources",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "ephemeral storage", "request"},
		AssessmentQueryApplicable: "target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.container.spec.request.ephemeral-storage.not-set IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: advice_common.ReadAdviceFile(EphemeralStorageRequestContent, "instructions.md"),
					Motivation:  advice_common.ReadAdviceFile(EphemeralStorageRequestContent, "motivation.md"),
					Summary:     advice_common.ReadAdviceFile(EphemeralStorageRequestContent, "action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: advice_common.ReadAdviceFile(EphemeralStorageRequestContent, "implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: advice_common.ReadAdviceFile(EphemeralStorageRequestContent, "validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               EphemeralStorageRequestID + ".validation.1",
						Type:             "TEXT",
						Name:             "Requested ephemeral storage resources are reasonable",
						ShortDescription: "I confirm that the requested ephemeral storage resources are reasonable to avoid overcommitment of resources and optimize scheduling of workload resources.",
						Description:      extutil.Ptr(advice_common.ReadAdviceFile(EphemeralStorageRequestContent, "validation_1_description.md")),
					},
				}),
			},
		},
	}
}
