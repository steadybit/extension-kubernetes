// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extconfig

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/advice-kit/go/advice_kit_sdk"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	advice_kit_sdk.AdviceConfig
	ClusterName                            string   `required:"true" split_words:"true"`
	LabelFilter                            []string `required:"false" split_words:"true" default:"controller-revision-hash,pod-template-generation,pod-template-hash"`
	AdviceSingleReplicaMinReplicas         int      `json:"adviceSingleReplicaMinReplicas" split_words:"true" required:"false" default:"2"`
	DisableDiscoveryExcludes               bool     `required:"false" split_words:"true" default:"false"`
	LogKubernetesHttpRequests              bool     `required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledArgoRollout           bool     `json:"discoveryDisabledArgoRollout" required:"false" split_words:"true" default:"true"`
	DiscoveryDisabledCluster               bool     `json:"discoveryDisabledCluster" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledContainer             bool     `json:"discoveryDisabledContainer" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledDaemonSet             bool     `json:"discoveryDisabledDaemonSet" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledDeployment            bool     `json:"discoveryDisabledDeployment" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledIngress               bool     `json:"discoveryDisabledIngress" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledNode                  bool     `json:"discoveryDisabledNode" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledPod                   bool     `json:"discoveryDisabledPod" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledReplicaSet            bool     `json:"discoveryDisabledReplicaSet" required:"false" split_words:"true" default:"false"`
	DiscoveryDisabledStatefulSet           bool     `json:"discoveryDisabledStatefulSet" required:"false" split_words:"true" default:"false"`
	DiscoveryAttributesExcludesContainer   []string `json:"discoveryAttributesExcludesContainer" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesDaemonSet   []string `json:"discoveryAttributesExcludesDaemonSet" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesDeployment  []string `json:"discoveryAttributesExcludesDeployment" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesIngress     []string `json:"discoveryAttributesExcludesIngress" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesNode        []string `json:"discoveryAttributesExcludesNode" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesPod         []string `json:"discoveryAttributesExcludesPod" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesReplicaSet  []string `json:"discoveryAttributesExcludesReplicaSet" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesStatefulSet []string `json:"discoveryAttributesExcludesStatefulSet" split_words:"true" required:"false"`
	DiscoveryMaxPodCount                   int      `json:"discoveryMaxPodCount" split_words:"true" required:"false" default:"50"`
	DiscoveryRefreshThrottle               int      `json:"DiscoveryRefreshThrottle" required:"false" split_words:"true" default:"20"`
	DiscoveryInformerResync                int      `json:"DiscoveryInformerResync" required:"false" split_words:"true" default:"600"`
	Namespace                              string   `json:"namespace" split_words:"true" required:"false" default:""`
	NginxDelaySkipImageCheck               bool     `json:"nginxDelaySkipImageCheck" split_words:"true" required:"false" default:"false"`
	PrintMemoryStatsInterval               int64    `json:"printMemoryStatsInterval" split_words:"true" required:"false" default:"0"`
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
	if Config.DisableDiscoveryExcludes {
		log.Info().Msg("Discovery excludes are disabled. Will also discover workloads labeled with steadybit.com/discovery-disabled=true.")
	}
}

func HasNamespaceFilter() bool {
	return Config.Namespace != ""
}
