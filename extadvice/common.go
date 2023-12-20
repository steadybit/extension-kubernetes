package extadvice

import (
	"embed"
	"github.com/rs/zerolog/log"
)

//go:embed cpu_limit/*
var CpuLimitContent embed.FS

//go:embed memory_limit/*
var MemoryLimitContent embed.FS

//go:embed ephemeral_storage_limit/*
var EphemeralStorageLimitContent embed.FS

//go:embed cpu_request/*
var CpuRequestContent embed.FS

//go:embed memory_request/*
var MemoryRequestContent embed.FS

//go:embed ephemeral_storage_request/*
var EphemeralStorageRequestContent embed.FS

//go:embed deployment_strategy/*
var DeploymentStrategyContent embed.FS

//go:embed image_latest_tag/*
var ImageVersioningContent embed.FS

//go:embed image_pull_policy/*
var ImagePullPolicyContent embed.FS

//go:embed horizontal_pod_autoscaler/*
var HorizontalPodAutoscalerContent embed.FS

//go:embed probes/*
var ProbesContent embed.FS

//go:embed single_replica/*
var SingleReplicaContent embed.FS

//go:embed host_podantiaffinity/*
var HostPodantiaffinityContent embed.FS

//go:embed single_aws_zone/*
var SingleAwsZoneContent embed.FS

//go:embed single_azure_zone/*
var SingleAzureZoneContent embed.FS

//go:embed single_gcp_zone/*
var SingleGcpZoneContent embed.FS

func ReadAdviceFile(fs embed.FS, fileName string) string {
	fileContent, err := fs.ReadFile(fileName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read file: %s", fileName)
	}
	return string(fileContent)
}
