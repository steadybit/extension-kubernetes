package advice

import (
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

func GetAdviceDescriptionImageVersioning(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Image Versioning",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "image", "versioning", "latest", "tag"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\" and k8s.kube-score.container-image-tag.grade IS PRESENT",
		AssessmentQueryActionNeeded: "k8s.kube-score.container-image-tag.grade != \"OK\" and k8s.kube-score.container-image-tag.grade != \"SKIPPED\"",
		Experiments:                 nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(ImageVersioningContent, "image_latest_tag/instructions.md"),
				Motivation:  ReadAdviceFile(ImageVersioningContent, "image_latest_tag/motivation.md"),
				Summary:     ReadAdviceFile(ImageVersioningContent, "image_latest_tag/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(ImageVersioningContent, "image_latest_tag/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(ImageVersioningContent, "image_latest_tag/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionImagePullPolicy(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Image Pull Policy",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "image", "pull", "policy"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\" and k8s.kube-score.container-image-pull-policy.grade IS PRESENT",
		AssessmentQueryActionNeeded: "k8s.kube-score.container-image-pull-policy.grade != \"OK\" and k8s.kube-score.container-image-pull-policy.grade != \"SKIPPED\"",
		Experiments:                 nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/instructions.md"),
				Motivation:  ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/motivation.md"),
				Summary:     ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(ImagePullPolicyContent, "image_pull_policy/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionDeploymentStrategy(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Deployment Strategy",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "strategy"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\"",
		AssessmentQueryActionNeeded: "k8s.deployment.strategy!=\"RollingUpdate\" ",
		Experiments:                 nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/instructions.md"),
				Motivation:  ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/motivation.md"),
				Summary:     ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(DeploymentStrategyContent, "deployment_strategy/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionHorizontalPodAutoscaler(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Horizontal Pod Autoscaler",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "horizontal pod autoscaler", "replicas"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\" and k8s.deployment.hpa.existent=\"true\"",
		AssessmentQueryActionNeeded: "k8s.deployment.replicas IS PRESENT",
		Experiments:                 nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(HorizontalPodAutoscalerContent, "horizontal_pod_autoscaler/instructions.md"),
				Motivation:  ReadAdviceFile(HorizontalPodAutoscalerContent, "horizontal_pod_autoscaler/motivation.md"),
				Summary:     ReadAdviceFile(HorizontalPodAutoscalerContent, "horizontal_pod_autoscaler/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(HorizontalPodAutoscalerContent, "horizontal_pod_autoscaler/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(HorizontalPodAutoscalerContent, "horizontal_pod_autoscaler/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionCPULimit(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "CPU Limit",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "cpu", "limit"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\"",
		AssessmentQueryActionNeeded: "k8s.container.spec.name.limit.cpu.not-set IS PRESENT",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         targetTypeName + ".k8s-cpu-limit.experiment-1",
			Name:       "CPU Overload",
			Description: extutil.Ptr("CPU limits are important to avoid unwanted side effects that can be triggered by increased CPU consumption of a single component. With the help of an experiment, a CPU overload can be simulated to check whether Kubernetes applies the configured limit correctly."),
			Experiment: ReadAdviceFile(CpuLimitContent, "cpu_limit/experiment_cpu_limit.json"),
		},
		},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(CpuLimitContent, "cpu_limit/instructions.md"),
				Motivation:  ReadAdviceFile(CpuLimitContent, "cpu_limit/motivation.md"),
				Summary:     ReadAdviceFile(CpuLimitContent, "cpu_limit/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(CpuLimitContent, "cpu_limit/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(CpuLimitContent, "cpu_limit/implemented.md"),
			},
		},
	}
}
func GetAdviceDescriptionSingleReplica(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Single Replica",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "replica", "pod"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\"",
		AssessmentQueryActionNeeded: "k8s.deployment.replicas IS NOT PRESENT OR k8s.deployment.replicas != \"1\"",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         targetType + ".k8s-single-replica.experiment-1",
			Name:       "Pod Failure",
			Description: extutil.Ptr("Pod failures and errors can occur repeatedly during operation. With an experiment these errors are simulated and you can check, whether the functionality of your application is still ensured."),
			Experiment: ReadAdviceFile(SingleReplicaContent, "single_replica/experiment_pod_failure.json"),
		},
		},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(SingleReplicaContent, "single_replica/instructions.md"),
				Motivation:  ReadAdviceFile(SingleReplicaContent, "single_replica/motivation.md"),
				Summary:     ReadAdviceFile(SingleReplicaContent, "single_replica/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(SingleReplicaContent, "single_replica/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(SingleReplicaContent, "single_replica/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionHostPodantiaffinity(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Pod Anti Affinity",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "host", "pod", "antiaffinity"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\" and k8s.kube-score.deployment-has-host-podantiaffinity.grade IS PRESENT",
		AssessmentQueryActionNeeded: "k8s.kube-score.deployment-has-host-podantiaffinity.grade != \"OK\" and k8s.kube-score.deployment-has-host-podantiaffinity.grade != \"SKIPPED\"",
		Experiments: nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/instructions.md"),
				Motivation:  ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/motivation.md"),
				Summary:     ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(HostPodantiaffinityContent, "host_podantiaffinity/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionLivenessProbe(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Liveness Probe",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "probes", "liveness"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\"",
		AssessmentQueryActionNeeded: "k8s.container.probes.liveness.not-set IS PRESENT",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         targetType + ".k8s-liveness-probe.experiment-1",
			Name:       "Pod Lifecycle",
			Description: extutil.Ptr("Liveness probes can help improve the availability of pods in a Kubernetes cluster when they are properly confugured and tested. By simulating slow-responding health endpoints as part of an experiment, it is possible to verify that Kubernetes correctly executes the Liveness Probe and restarts the failing pod."),
			Experiment: ReadAdviceFile(LivenessProbeContent, "liveness_probe/experiment_pod_lifecycle.json"),
		},
		},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(LivenessProbeContent, "liveness_probe/instructions.md"),
				Motivation:  ReadAdviceFile(LivenessProbeContent, "liveness_probe/motivation.md"),
				Summary:     ReadAdviceFile(LivenessProbeContent, "liveness_probe/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(LivenessProbeContent, "liveness_probe/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(LivenessProbeContent, "liveness_probe/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionReadinessProbe(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Readiness Probe",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "probes", "readiness"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\"",
		AssessmentQueryActionNeeded: "k8s.container.probes.readiness.not-set IS PRESENT",
		Experiments: nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(ReadinessProbeContent, "readiness_probe/instructions.md"),
				Motivation:  ReadAdviceFile(ReadinessProbeContent, "readiness_probe/motivation.md"),
				Summary:     ReadAdviceFile(ReadinessProbeContent, "readiness_probe/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(ReadinessProbeContent, "readiness_probe/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(ReadinessProbeContent, "readiness_probe/implemented.md"),
			},
		},
	}
}


func GetAdviceDescriptionMemoryLimit(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Memory Limit",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "memory", "limit"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\"",
		AssessmentQueryActionNeeded: "k8s.container.spec.name.limit.memory.not-set IS PRESENT",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         targetType + ".k8s-memory-limit.experiment-1",
			Name:       "Memory Overload",
			Description: extutil.Ptr("Memory limits are important to avoid unwanted side effects that can be triggered by increased memory consumption of a single component. With the help of an experiment, a memory overload can be simulated to check whether Kubernetes applies the configured limit correctly."),
			Experiment: ReadAdviceFile(MemoryLimitContent, "memory_limit/experiment_memory_limit.json"),
		},
		},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(MemoryLimitContent, "memory_limit/instructions.md"),
				Motivation:  ReadAdviceFile(MemoryLimitContent, "memory_limit/motivation.md"),
				Summary:     ReadAdviceFile(MemoryLimitContent, "memory_limit/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(MemoryLimitContent, "memory_limit/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(MemoryLimitContent, "memory_limit/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionSingleAwsZone(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "AWS Host Scheduling",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "aws", "zone"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\" AND aws.zone IS PRESENT",
		AssessmentQueryActionNeeded: "aws.zone HAS ONE VALUE",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         targetType + ".single-aws-zone.experiment-1",
			Name:       "Zone Outage",
			Description: extutil.Ptr("It is recommended to always split its components into different zones so that in case of a failure of one zone, the system still works well. An experiment can be used to validate whether the system can handle the failure of an entire zone."),
			Experiment: ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/experiment_zone_outage.json"),
		},
		},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/instructions.md"),
				Motivation:  ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/motivation.md"),
				Summary:     ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(SingleAwsZoneContent, "single_aws_zone/implemented.md"),
			},
		},
	}
}

func GetAdviceDescriptionSingleAzureZone(id string, targetType string, targetTypeName string) advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          id,
		Label:                       "Azure Host Scheduling",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", targetTypeName, "azure", "zone"},
		AssessmentQueryApplicable:   "target.type=\"" + targetType + "\" AND azure.zone IS PRESENT",
		AssessmentQueryActionNeeded: "azure.zone HAS ONE VALUE",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         targetType + ".single-azure-zone.experiment-1",
			Name:       "Zone Outage",
			Description: extutil.Ptr("It is recommended to always split its components into different zones so that in case of a failure of one zone, the system still works well. An experiment can be used to validate whether the system can handle the failure of an entire zone."),
			Experiment: ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/experiment_zone_outage.json"),
		},
		},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/instructions.md"),
				Motivation:  ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/motivation.md"),
				Summary:     ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: ReadAdviceFile(SingleAzureZoneContent, "single_azure_zone/implemented.md"),
			},
		},
	}
}