// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extdeployment

import (
	"embed"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
)

const DeploymentStrategyID = DeploymentTargetType + ".advice.k8s-deployment-strategy"
const CpuLimitID = DeploymentTargetType + ".advice.k8s-cpu-limit"
const MemoryLimitID = DeploymentTargetType + ".advice.k8s-memory-limit"
const HorizontalPodAutoscalerID = DeploymentTargetType + ".advice.k8s-horizontal-pod-autoscaler"
const ImageVersioningID = DeploymentTargetType + ".advice.k8s-image-latest-tag"

func RegisterDeploymentAdviceHandlers() {
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-deployment-strategy", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionDeploymentStrategy))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-cpu-limit", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionCPULimit))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-memory-limit", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionMemoryLimit))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-horizontal-pod-autoscaler", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionHorizontalPodAutoscaler))
	exthttp.RegisterHttpHandler("/deployment/advice/k8s-image-latest-tag", exthttp.GetterAsHandler(getDeploymentAdviceDescriptionImageVersioning))
}

//go:embed advice_templates/cpu_limit/*
var cpuLimitContent embed.FS

//go:embed advice_templates/memory_limit/*
var memoryLimitContent embed.FS

//go:embed advice_templates/deployment_strategy/*
var deploymentStrategyContent embed.FS

//go:embed advice_templates/image_latest_tag/*
var imageVersioningContent embed.FS

//go:embed advice_templates/horizontal_pod_autoscaler/*
var horizontalPodAutoscalerContent embed.FS

func getDeploymentAdviceDescriptionImageVersioning() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          ImageVersioningID,
		Label:                       "Image Versioning",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", "deployment", "image", "versioning", "latest", "tag"},
		AssessmentQueryApplicable:   "target.type=\"" + DeploymentTargetType + "\"",
		AssessmentQueryActionNeeded: "k8s.container.image.with-latest-tag IS PRESENT",
		Experiments:                 nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: readLocalFile(imageVersioningContent, "advice_templates/image_latest_tag/instructions.md"),
				Motivation:  readLocalFile(imageVersioningContent, "advice_templates/image_latest_tag/motivation.md"),
				Summary:     readLocalFile(imageVersioningContent, "advice_templates/image_latest_tag/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: readLocalFile(imageVersioningContent, "advice_templates/image_latest_tag/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: readLocalFile(imageVersioningContent, "advice_templates/image_latest_tag/implemented.md"),
			},
		},
	}
}

func getDeploymentAdviceDescriptionDeploymentStrategy() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          DeploymentStrategyID,
		Label:                       "Deployment Strategy",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", "deployment", "strategy"},
		AssessmentQueryApplicable:   "target.type=\"" + DeploymentTargetType + "\"",
		AssessmentQueryActionNeeded: "k8s.deployment.strategy!=\"RollingUpdate\" ",
		Experiments:                 nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: readLocalFile(deploymentStrategyContent, "advice_templates/deployment_strategy/instructions.md"),
				Motivation:  readLocalFile(deploymentStrategyContent, "advice_templates/deployment_strategy/motivation.md"),
				Summary:     readLocalFile(deploymentStrategyContent, "advice_templates/deployment_strategy/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: readLocalFile(deploymentStrategyContent, "advice_templates/deployment_strategy/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: readLocalFile(deploymentStrategyContent, "advice_templates/deployment_strategy/implemented.md"),
			},
		},
	}
}

func getDeploymentAdviceDescriptionHorizontalPodAutoscaler() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          HorizontalPodAutoscalerID,
		Label:                       "Horizontal Pod Autoscaler",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", "deployment", "horizontal pod autoscaler", "replicas"},
		AssessmentQueryApplicable:   "target.type=\"" + DeploymentTargetType + "\" and k8s.deployment.hpa.existent=\"true\"",
		AssessmentQueryActionNeeded: "k8s.deployment.replicas IS PRESENT",
		Experiments:                 nil,
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: readLocalFile(horizontalPodAutoscalerContent, "advice_templates/horizontal_pod_autoscaler/instructions.md"),
				Motivation:  readLocalFile(horizontalPodAutoscalerContent, "advice_templates/horizontal_pod_autoscaler/motivation.md"),
				Summary:     readLocalFile(horizontalPodAutoscalerContent, "advice_templates/horizontal_pod_autoscaler/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: readLocalFile(horizontalPodAutoscalerContent, "advice_templates/horizontal_pod_autoscaler/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: readLocalFile(horizontalPodAutoscalerContent, "advice_templates/horizontal_pod_autoscaler/implemented.md"),
			},
		},
	}
}

func getDeploymentAdviceDescriptionCPULimit() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          CpuLimitID,
		Label:                       "CPU Limit",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", "deployment", "cpu", "limit"},
		AssessmentQueryApplicable:   "target.type=\"" + DeploymentTargetType + "\"",
		AssessmentQueryActionNeeded: "k8s.container.spec.name.limit.cpu.not-set IS PRESENT",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         DeploymentTargetType + ".advice.k8s-cpu-limit.experiment-1",
			Name:       "CPU Overload",
			Experiment: readLocalFile(cpuLimitContent, "advice_templates/cpu_limit/experiment_cpu_limit.json"),
		},
		},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: readLocalFile(cpuLimitContent, "advice_templates/cpu_limit/instructions.md"),
				Motivation:  readLocalFile(cpuLimitContent, "advice_templates/cpu_limit/motivation.md"),
				Summary:     readLocalFile(cpuLimitContent, "advice_templates/cpu_limit/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: readLocalFile(cpuLimitContent, "advice_templates/cpu_limit/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: readLocalFile(cpuLimitContent, "advice_templates/cpu_limit/implemented.md"),
			},
		},
	}
}


func getDeploymentAdviceDescriptionMemoryLimit() advice_kit_api.AdviceDefinition {
	return advice_kit_api.AdviceDefinition{
		Id:                          MemoryLimitID,
		Label:                       "Memory Limit",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M11.9436%207.04563C12.1262%206.98477%2012.3235%206.98477%2012.5061%207.04563L17.8407%208.82395C18.2037%208.94498%2018.4486%209.28468%2018.4485%209.66728C18.4485%2010.0499%2018.2036%2010.3895%2017.8405%2010.5105L12.5059%2012.2877C12.3235%2012.3485%2012.1262%2012.3485%2011.9438%2012.2877L6.60918%2010.5105C6.24611%2010.3895%206.00119%2010.0499%206.00116%209.66728C6.00112%209.28468%206.24598%208.94498%206.60902%208.82395L11.9436%207.04563Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M7.20674%2013.2736C6.68268%2013.0989%206.11622%2013.3821%205.94153%2013.9062C5.76684%2014.4302%206.05007%2014.9967%206.57414%2015.1714L11.9087%2016.9496C12.114%2017.018%2012.336%2017.018%2012.5413%2016.9496L17.8759%2015.1714C18.4%2014.9967%2018.6832%2014.4302%2018.5085%2013.9062C18.3338%2013.3821%2017.7674%2013.0989%2017.2433%2013.2736L12.225%2014.9463L7.20674%2013.2736Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M11.6491%201.06354C11.8754%200.97882%2012.1246%200.97882%2012.3509%201.06354L22.3506%204.80836C22.7412%204.95463%2023%205.32784%2023%205.74482V18.2552C23%2018.6722%2022.7412%2019.0454%2022.3506%2019.1916L12.3509%2022.9365C12.1246%2023.0212%2011.8754%2023.0212%2011.6491%2022.9365L1.64938%2019.1916C1.2588%2019.0454%201%2018.6722%201%2018.2552V5.74482C1%205.32784%201.2588%204.95463%201.64938%204.80836L11.6491%201.06354ZM3.00047%206.43809V17.5619L12%2020.9321L20.9995%2017.5619V6.43809L12%203.06785L3.00047%206.43809Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A",
		Tags:                        &[]string{"kubernetes", "deployment", "memory", "limit"},
		AssessmentQueryApplicable:   "target.type=\"" + DeploymentTargetType + "\"",
		AssessmentQueryActionNeeded: "k8s.container.spec.name.limit.memory.not-set IS PRESENT",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         DeploymentTargetType + ".advice.k8s-memory-limit.experiment-1",
			Name:       "Memory Overload",
			Experiment: readLocalFile(memoryLimitContent, "advice_templates/memory_limit/experiment_memory_limit.json"),
		},
		},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: readLocalFile(memoryLimitContent, "advice_templates/memory_limit/instructions.md"),
				Motivation:  readLocalFile(memoryLimitContent, "advice_templates/memory_limit/motivation.md"),
				Summary:     readLocalFile(memoryLimitContent, "advice_templates/memory_limit/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: readLocalFile(memoryLimitContent, "advice_templates/memory_limit/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: readLocalFile(memoryLimitContent, "advice_templates/memory_limit/implemented.md"),
			},
		},
	}
}

func readLocalFile(fs embed.FS, fileName string) string {
	fileContent, err := fs.ReadFile(fileName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read file: %s", fileName)
	}
	return string(fileContent)
}
