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
	ClusterName                           string   `required:"true" split_words:"true"`
	LabelFilter                           []string `required:"false" split_words:"true" default:"controller-revision-hash,pod-template-generation,pod-template-hash"`
	DisableDiscoveryExcludes              bool     `required:"false" split_words:"true" default:"false"`
	DiscoveryAttributesExcludesContainer  []string `json:"discoveryAttributesExcludesContainer" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesDeployment []string `json:"discoveryAttributesExcludesDeployment" split_words:"true" required:"false"`
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
