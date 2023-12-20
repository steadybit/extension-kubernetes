package extadvice

import (
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/extdaemonset"
	"github.com/steadybit/extension-kubernetes/extdeployment"
	"github.com/steadybit/extension-kubernetes/extstatefulset"
)

const DeploymentStrategyID = "com.steadybit.extension_kubernetes.advice.k8s-deployment-strategy"
const CpuLimitID = "com.steadybit.extension_kubernetes.advice.k8s-cpu-limit"
const MemoryLimitID = "com.steadybit.extension_kubernetes.advice.k8s-memory-limit"
const EphemeralStorageLimitID = "com.steadybit.extension_kubernetes.advice.k8s-ephemeral-storage-limit"
const CpuRequestID = "com.steadybit.extension_kubernetes.advice.k8s-cpu-request"
const MemoryRequestID = "com.steadybit.extension_kubernetes.advice.k8s-memory-request"
const EphemeralStorageRequestID = "com.steadybit.extension_kubernetes.advice.k8s-ephemeral-storage-request"
const ImageVersioningID = "com.steadybit.extension_kubernetes.advice.k8s-image-latest-tag"
const ImagePullPolicyID = "com.steadybit.extension_kubernetes.advice.k8s-image-pull-policy"
const ProbesID = "com.steadybit.extension_kubernetes.advice.k8s-probes"
const SingleReplicaID = "com.steadybit.extension_kubernetes.advice.k8s-single-replica"
const HostPodantiaffinityID = "com.steadybit.extension_kubernetes.advice.k8s-host-podantiaffinity"
const SingleAWSZoneID = "com.steadybit.extension_kubernetes.advice.single-aws-zone"
const SingleAzureZoneID = "com.steadybit.extension_kubernetes.advice.single-azure-zone"
const SingleGCPZoneID = "com.steadybit.extension_kubernetes.advice.single-gcp-zone"

func RegisterAdviceHandlers() {
	exthttp.RegisterHttpHandler("/advice/k8s-deployment-strategy", exthttp.GetterAsHandler(GetAdviceDescriptionDeploymentStrategy))
	exthttp.RegisterHttpHandler("/advice/k8s-cpu-limit", exthttp.GetterAsHandler(GetAdviceDescriptionCPULimit))
	exthttp.RegisterHttpHandler("/advice/k8s-memory-limit", exthttp.GetterAsHandler(GetAdviceDescriptionMemoryLimit))
	exthttp.RegisterHttpHandler("/advice/k8s-ephemeral-storage-limit", exthttp.GetterAsHandler(GetAdviceDescriptionEphemeralStorageLimit))
	exthttp.RegisterHttpHandler("/advice/k8s-cpu-request", exthttp.GetterAsHandler(GetAdviceDescriptionCPURequest))
	exthttp.RegisterHttpHandler("/advice/k8s-memory-request", exthttp.GetterAsHandler(GetAdviceDescriptionMemoryRequest))
	exthttp.RegisterHttpHandler("/advice/k8s-ephemeral-storage-request", exthttp.GetterAsHandler(GetAdviceDescriptionEphemeralStorageRequest))
	exthttp.RegisterHttpHandler("/advice/k8s-image-latest-tag", exthttp.GetterAsHandler(GetAdviceDescriptionImageVersioning))
	exthttp.RegisterHttpHandler("/advice/k8s-image-pull-policy", exthttp.GetterAsHandler(GetAdviceDescriptionImagePullPolicy))
	exthttp.RegisterHttpHandler("/advice/k8s-probes", exthttp.GetterAsHandler(GetAdviceDescriptionProbes))
	exthttp.RegisterHttpHandler("/advice/k8s-single-replica", exthttp.GetterAsHandler(GetAdviceDescriptionSingleReplica))
	exthttp.RegisterHttpHandler("/advice/k8s-host-podantiaffinity", exthttp.GetterAsHandler(GetAdviceDescriptionHostPodantiaffinity))
	exthttp.RegisterHttpHandler("/advice/single-aws-zone", exthttp.GetterAsHandler(GetAdviceDescriptionSingleAwsZone))
	exthttp.RegisterHttpHandler("/advice/single-azure-zone", exthttp.GetterAsHandler(GetAdviceDescriptionSingleAzureZone))
	exthttp.RegisterHttpHandler("/advice/single-gcp-zone", exthttp.GetterAsHandler(GetAdviceDescriptionSingleGcpZone))
}

func GetAdviceDescriptionImageVersioning() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                        ImageVersioningID,
		Label:                     "Image Version Explicitly Configured",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "image", "versioning", "latest", "tag"},
		AssessmentQueryApplicable: "target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.container.image.with-latest-tag IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(ImageVersioningContent, "image_latest_tag/instructions.md"),
					Motivation:  ReadAdviceFile(ImageVersioningContent, "image_latest_tag/motivation.md"),
					Summary:     ReadAdviceFile(ImageVersioningContent, "image_latest_tag/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(ImageVersioningContent, "image_latest_tag/implemented.md"),
				},
			},
		},
	}
}

func GetAdviceDescriptionImagePullPolicy() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                        ImagePullPolicyID,
		Label:                     "Image Pull Policy Set To Always",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "image", "pull", "policy"},
		AssessmentQueryApplicable: "target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.container.image.without-image-pull-policy-always IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/instructions.md"),
					Motivation:  ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/motivation.md"),
					Summary:     ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/implemented.md"),
				},
			},
		},
	}
}

func GetAdviceDescriptionDeploymentStrategy() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                        DeploymentStrategyID,
		Label:                     "Rolling Update Deployments",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "deployment", "strategy"},
		AssessmentQueryApplicable: "target.type=\"" + extdeployment.DeploymentTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.deployment.strategy IS PRESENT AND k8s.deployment.strategy!=\"RollingUpdate\" ",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/instructions.md"),
					Motivation:  ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/motivation.md"),
					Summary:     ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/implemented.md"),
				},
			},
		},
	}
}

func GetAdviceDescriptionCPULimit() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        CpuLimitID,
		Label:                     "Limit CPU Resources",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "cpu", "limit"},
		AssessmentQueryApplicable: "target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.container.spec.limit.cpu.not-set IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(CpuLimitContent, "cpu_limit/instructions.md"),
					Motivation:  ReadAdviceFile(CpuLimitContent, "cpu_limit/motivation.md"),
					Summary:     ReadAdviceFile(CpuLimitContent, "cpu_limit/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(CpuLimitContent, "cpu_limit/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(CpuLimitContent, "cpu_limit/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               "com.steadybit.extension_kubernetes.k8s-cpu-limit.experiment-1",
						Type:             "EXPERIMENT",
						Name:             "CPU Overload",
						ShortDescription: "Check how ${target.steadybit.label} behaves when running at the CPU limit and whether your remaining Kubernetes resources at the host function properly.",
						Experiment:       extutil.Ptr(advice_kit_api.Experiment(ReadAdviceFile(CpuLimitContent, "cpu_limit/experiment_cpu_limit.json"))),
					},
				}),
			},
		},
	}
}

func GetAdviceDescriptionCPURequest() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        CpuRequestID,
		Label:                     "Requesting Reasonable CPU Resources",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "cpu", "request"},
		AssessmentQueryApplicable: "target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.container.spec.request.cpu.not-set IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(CpuRequestContent, "cpu_request/instructions.md"),
					Motivation:  ReadAdviceFile(CpuRequestContent, "cpu_request/motivation.md"),
					Summary:     ReadAdviceFile(CpuRequestContent, "cpu_request/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(CpuRequestContent, "cpu_request/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(CpuRequestContent, "cpu_request/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               CpuRequestID + ".validation.1",
						Type:             "TEXT",
						Name:             "Requested CPU resources are reasonable",
						ShortDescription: "I confirm that the requested CPU resources are reasonable to avoid overcommitment of resources and optimize scheduling of workload resources.",
						Description:      extutil.Ptr(ReadAdviceFile(CpuRequestContent, "cpu_request/validation_1_description.md")),
					},
				}),
			},
		},
	}
}

func GetAdviceDescriptionSingleReplica() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        SingleReplicaID,
		Label:                     "Redundant Pod Deployment",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "deployment", "statefulset", "replica", "pod"},
		AssessmentQueryApplicable: "target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.deployment.replicas IS NOT PRESENT OR k8s.deployment.replicas = \"1\"",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(SingleReplicaContent, "single_replica/instructions.md"),
					Motivation:  ReadAdviceFile(SingleReplicaContent, "single_replica/motivation.md"),
					Summary:     ReadAdviceFile(SingleReplicaContent, "single_replica/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(SingleReplicaContent, "single_replica/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(SingleReplicaContent, "single_replica/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               "com.steadybit.extension_kubernetes.k8s-single-replica.experiment-1",
						Type:             "EXPERIMENT",
						Name:             "Single Pod Failure",
						ShortDescription: "In case one pod of ${target.steadybit.label} fails, Kubernetes manages this accordingly by routing the traffic to the other pods within expected failure rates.",
						Experiment:       extutil.Ptr(advice_kit_api.Experiment(ReadAdviceFile(SingleReplicaContent, "single_replica/experiment_pod_failure.json"))),
					},
				}),
			},
		},
	}
}

func GetAdviceDescriptionHostPodantiaffinity() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        HostPodantiaffinityID,
		Label:                     "Schedule Pods on Different Hosts",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "deployment", "statefulset", "host", "pod", "antiaffinity"},
		AssessmentQueryApplicable: "(target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\") AND k8s.specification.has-host-podantiaffinity IS PRESENT",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.specification.has-host-podantiaffinity=\"false\"",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/instructions.md"),
					Motivation:  ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/motivation.md"),
					Summary:     ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/implemented.md"),
				},
			},
		},
	}
}

func GetAdviceDescriptionProbes() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        ProbesID,
		Label:                     "Probes Configured",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "probes", "liveness", "readiness"},
		AssessmentQueryApplicable: "(target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\") AND k8s.specification.probes.summary IS PRESENT",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.specification.probes.summary!=\"OK\"",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(ProbesContent, "probes/instructions.md"),
					Motivation:  ReadAdviceFile(ProbesContent, "probes/motivation.md"),
					Summary:     ReadAdviceFile(ProbesContent, "probes/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(ProbesContent, "probes/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(ProbesContent, "probes/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               "com.steadybit.extension_kubernetes.k8s-probes.experiment-1",
						Type:             "EXPERIMENT",
						Name:             "Container's Unhealthiness Is Detected Within Expected Failure Rates",
						ShortDescription: "Verify that Kubernetes can detect an unhealthy container of gateway, restarts the container, and routes traffic afterward within the expected failure rates.",
						Experiment:       extutil.Ptr(advice_kit_api.Experiment(ReadAdviceFile(ProbesContent, "probes/experiment_pod_lifecycle.json"))),
					},
				}),
			},
		},
	}
}

func GetAdviceDescriptionMemoryLimit() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        MemoryLimitID,
		Label:                     "Limit Memory Resources",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "memory", "limit"},
		AssessmentQueryApplicable: "target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.container.spec.limit.memory.not-set IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(MemoryLimitContent, "memory_limit/instructions.md"),
					Motivation:  ReadAdviceFile(MemoryLimitContent, "memory_limit/motivation.md"),
					Summary:     ReadAdviceFile(MemoryLimitContent, "memory_limit/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(MemoryLimitContent, "memory_limit/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(MemoryLimitContent, "memory_limit/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               "com.steadybit.extension_kubernetes.k8s-memory-limit.experiment-1",
						Type:             "EXPERIMENT",
						Name:             "Memory Overload",
						ShortDescription: "Check how ${target.steadybit.label} behaves when running at the memory limit and whether your remaining Kubernetes resources at the host function properly.",
						Experiment:       extutil.Ptr(advice_kit_api.Experiment(ReadAdviceFile(MemoryLimitContent, "memory_limit/experiment_memory_limit.json"))),
					},
				}),
			},
		},
	}
}

func GetAdviceDescriptionMemoryRequest() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        MemoryRequestID,
		Label:                     "Requesting Reasonable Memory Resources",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "memory", "request"},
		AssessmentQueryApplicable: "target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.container.spec.request.memory.not-set IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(MemoryRequestContent, "memory_request/instructions.md"),
					Motivation:  ReadAdviceFile(MemoryRequestContent, "memory_request/motivation.md"),
					Summary:     ReadAdviceFile(MemoryRequestContent, "memory_request/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(MemoryRequestContent, "memory_request/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(MemoryRequestContent, "memory_request/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               MemoryRequestID + ".validation.1",
						Type:             "TEXT",
						Name:             "Requested Memory resources are reasonable",
						ShortDescription: "I confirm that the requested memory resources are reasonable to avoid overcommitment of resources and optimize scheduling of workload resources.",
						Description:      extutil.Ptr(ReadAdviceFile(MemoryRequestContent, "memory_request/validation_1_description.md")),
					},
				}),
			},
		},
	}
}

func GetAdviceDescriptionEphemeralStorageLimit() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        EphemeralStorageLimitID,
		Label:                     "Limit Ephemeral Storage Resources",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "ephemeral storage", "limit"},
		AssessmentQueryApplicable: "target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\"",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "k8s.container.spec.limit.ephemeral-storage.not-set IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(EphemeralStorageLimitContent, "ephemeral_storage_limit/instructions.md"),
					Motivation:  ReadAdviceFile(EphemeralStorageLimitContent, "ephemeral_storage_limit/motivation.md"),
					Summary:     ReadAdviceFile(EphemeralStorageLimitContent, "ephemeral_storage_limit/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(EphemeralStorageLimitContent, "ephemeral_storage_limit/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(MemoryLimitContent, "ephemeral_storage_limit/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               "com.steadybit.extension_kubernetes.k8s-ephemeral-storage-limit.experiment-1",
						Type:             "EXPERIMENT",
						Name:             "Ephemeral Storage Overload",
						ShortDescription: "Check how ${target.steadybit.label} behaves when exceeding the ephemeral storage limit and whether your remaining Kubernetes resources at the host function properly.",
						Experiment:       extutil.Ptr(advice_kit_api.Experiment(ReadAdviceFile(MemoryLimitContent, "ephemeral_storage_limit/experiment_ephemeral_storage_limit.json"))),
					},
				}),
			},
		},
	}
}

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
					Instruction: ReadAdviceFile(EphemeralStorageRequestContent, "ephemeral_storage_request/instructions.md"),
					Motivation:  ReadAdviceFile(EphemeralStorageRequestContent, "ephemeral_storage_request/motivation.md"),
					Summary:     ReadAdviceFile(EphemeralStorageRequestContent, "ephemeral_storage_request/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(EphemeralStorageRequestContent, "ephemeral_storage_request/implemented.md"),
				},
			},
		},
	}
}

func GetAdviceDescriptionSingleAwsZone() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        SingleAWSZoneID,
		Label:                     "Schedule Pods Across AWS Zones",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "aws", "zone"},
		AssessmentQueryApplicable: "(target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\") AND aws.zone IS PRESENT",

		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "count(aws.zone) = 1",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/instructions.md"),
					Motivation:  ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/motivation.md"),
					Summary:     ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               "com.steadybit.extension_kubernetes.single-aws-zone.experiment-1",
						Type:             "EXPERIMENT",
						Name:             "Availability Zone Outage",
						ShortDescription: "When a single AWS availability zone fails, there are still pods of ${target.steadybit.label} ready to continue providing offered features.",
						Experiment:       extutil.Ptr(advice_kit_api.Experiment(ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/experiment_zone_outage.json"))),
					},
				}),
			},
		},
	}
}
func GetAdviceDescriptionSingleGcpZone() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        SingleGCPZoneID,
		Label:                     "Schedule Pods Across GCP Zones",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "gcp", "zone"},
		AssessmentQueryApplicable: "(target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\") AND gcp.zone IS PRESENT",

		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "count(gcp.zone) = 1",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(SingleGcpZoneContent, "single_gcp_zone/instructions.md"),
					Motivation:  ReadAdviceFile(SingleGcpZoneContent, "single_gcp_zone/motivation.md"),
					Summary:     ReadAdviceFile(SingleGcpZoneContent, "single_gcp_zone/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(SingleGcpZoneContent, "single_gcp_zone/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(SingleGcpZoneContent, "single_gcp_zone/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               "com.steadybit.extension_kubernetes.single-gcp-zone.experiment-1",
						Type:             "EXPERIMENT",
						Name:             "Availability Zone Outage",
						ShortDescription: "When a single GCP availability zone fails, there are still pods of ${target.steadybit.label} ready to continue providing offered features.",
						Experiment:       extutil.Ptr(advice_kit_api.Experiment(ReadAdviceFile(SingleGcpZoneContent, "single_gcp_zone/experiment_zone_outage.json"))),
					},
				}),
			},
		},
	}
}

func GetAdviceDescriptionSingleAzureZone() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                        SingleAzureZoneID,
		Label:                     "Scheduling Pods Across Azure Zones",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                      &[]string{"kubernetes", "daemonset", "deployment", "statefulset", "azure", "zone"},
		AssessmentQueryApplicable: "(target.type=\"" + extdaemonset.DaemonSetTargetType + "\" OR target.type=\"" + extdeployment.DeploymentTargetType + "\" OR target.type=\"" + extstatefulset.StatefulSetTargetType + "\") AND azure.zone IS PRESENT",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "count(azure.zone) = 1",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/instructions.md"),
					Motivation:  ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/motivation.md"),
					Summary:     ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:               "com.steadybit.extension_kubernetes.single-azure-zone.experiment-1",
						Type:             "EXPERIMENT",
						Name:             "Availability Zone Outage",
						ShortDescription: "When a single Azure availability zone fails, there are still pods of ${target.steadybit.label} ready to continue providing offered features.",
						Experiment:       extutil.Ptr(advice_kit_api.Experiment(ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/experiment_zone_outage.json"))),
					},
				}),
			},
		},
	}
}
