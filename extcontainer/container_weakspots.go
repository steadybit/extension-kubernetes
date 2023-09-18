// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcontainer

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/weakspot-kit/go/weakspot_kit_api"
	"os"
)

func RegisterContainerWeakspotHandlers() {
	exthttp.RegisterHttpHandler("/container/weakspots/k8s-cpu-limit", exthttp.GetterAsHandler(getContainerWeakspotDescriptionCPULimit))
}

func getContainerWeakspotDescriptionCPULimit() weakspot_kit_api.WeakspotDescription {
	experimentTemplateCpuLimit, err := os.ReadFile("./extcontainer/weakspot_templates/cpu_limit/experiment_cpu_limit.json")
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read experiment template file: %s", "./extcontainer/weakspot_templates/experiment_cpu_limit.json")
	}

	finding, err := os.ReadFile("./extcontainer/weakspot_templates/cpu_limit/finding.md")
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read finding template file: %s", "./extcontainer/weakspot_templates/cpu_limit/finding.md")
	}

	guidance, err := os.ReadFile("./extcontainer/weakspot_templates/cpu_limit/guidance.md")
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read guidance template file: %s", "./extcontainer/weakspot_templates/cpu_limit/guidance.md")
	}

	looksgood, err := os.ReadFile("./extcontainer/weakspot_templates/cpu_limit/looksgood.md")
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read looksgood template file: %s", "./extcontainer/weakspot_templates/cpu_limit/looksgood.md")
	}

	instructions, err := os.ReadFile("./extcontainer/weakspot_templates/cpu_limit/instructions.md")
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read instructions template file: %s", "./extcontainer/weakspot_templates/cpu_limit/instructions.md")
	}

	return weakspot_kit_api.WeakspotDescription{
		Id:         KubernetesContainerEnrichmentDataType + "weakspot.k8s-cpu-limit",
		Label:     "CPU Limit",
		Version:    extbuild.GetSemverVersionStringOrUnknown(),
		Icon:       kubernetesContainerIcon,
		Tags:      &[]string{"kubernetes", "container", "cpu", "limit"},
		AssesmentBaseQuery: "target.type=\"com.steadybit.extension_container.container\" and k8s.container.ready=\"true\"",
		AssesmentQueryAddon: "k8s.container.cpu.limit=\"0\"",
		Experiments: &[]weakspot_kit_api.Experiment{
			string(experimentTemplateCpuLimit),
		},
		Finding:     extutil.Ptr(string(finding)),
		Guidance:     extutil.Ptr(string(guidance)),
		LooksGood:    extutil.Ptr(string(looksgood)),
		Instructions:    extutil.Ptr(string(instructions)),
	}
}

