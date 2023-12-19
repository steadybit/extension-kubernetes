// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extconfig

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	ClusterName                                 string   `required:"true" split_words:"true"`
	LabelFilter                                 []string `required:"false" split_words:"true" default:"controller-revision-hash,pod-template-generation,pod-template-hash"`
	ActiveAdviceList                            []string `required:"false" split_words:"true" default:"*"`
	DisableDiscoveryExcludes                    bool     `required:"false" split_words:"true" default:"false"`
	LogKubernetesHttpRequests                   bool     `required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledContainer                  bool     `json:"discoveryDisabledContainer" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledDeployment                 bool     `json:"discoveryDisabledDeployment" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledStatefulSet                bool     `json:"discoveryDisabledStatefulSet" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledDaemonSet                  bool     `json:"discoveryDisabledDaemonSet" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledPod                        bool     `json:"discoveryDisabledPod" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledNode                       bool     `json:"discoveryDisabledNode" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledCluster                    bool     `json:"discoveryDisabledCluster" required:"false" split_words:"true" default:"false"`
	DiscoveryAttributesExcludesContainer        []string `json:"discoveryAttributesExcludesContainer" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesDeployment       []string `json:"discoveryAttributesExcludesDeployment" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesStatefulSet      []string `json:"discoveryAttributesExcludesStatefulSet" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesDaemonSet        []string `json:"discoveryAttributesExcludesDaemonSet" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesPod              []string `json:"discoveryAttributesExcludesPod" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesNode             []string `json:"discoveryAttributesExcludesNode" split_words:"true" required:"false"`
	DiscoveryMaxPodCount                        int      `json:"discoveryMaxPodCount" split_words:"true" required:"false" default:"50"`
	AdviceIgnoreContainerCpuLimitRequirement    bool     `json:"adviceIgnoreContainerCpuLimitRequirement" split_words:"true" required:"false" default:"false"`
	AdviceIgnoreContainerMemoryLimitRequirement bool     `json:"adviceIgnoreContainerMemoryLimitRequirement" split_words:"true" required:"false" default:"false"`
}

var (
	Config Specification
)

func ParseConfiguration() {
	err := envconfig.Process("steadybit_extension", &Config)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse configuration from environment.")
	}
}

func ValidateConfiguration() {
	// You may optionally validate the configuration here.
}
