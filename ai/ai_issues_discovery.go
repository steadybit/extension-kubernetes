/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package ai

import (
	"context"
	"strings"
	"time"

	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

const (
	ReliabilityIssueTargetType = "com.steadybit.extension_kubernetes.ai.reliability-issues"
	clusterAttribute           = "k8s.ai.reliability_issues.cluster"
	namespaceAttribute         = "k8s.ai.reliability_issues.namespace"
	kindAttribute              = "k8s.ai.reliability_issues.kind"
	nameAttribute              = "k8s.ai.reliability_issues.name"
	titleAttribute             = "k8s.ai.reliability_issues.title"
)

type reliabilityIssueDiscovery struct {
}

var (
	_ discovery_kit_sdk.TargetDescriber    = (*reliabilityIssueDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber = (*reliabilityIssueDiscovery)(nil)
)

// NewReliabilityIssueDiscovery creates the cached discovery, refreshing every minute.
func NewReliabilityIssueDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &reliabilityIssueDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 1*time.Minute),
	)
}

func (d *reliabilityIssueDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: ReliabilityIssueTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *reliabilityIssueDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:      ReliabilityIssueTargetType,
		Label:   discovery_kit_api.PluralLabel{One: "Kubernetes AI reliability issue", Other: "Kubernetes AI reliability issues"},
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		// Category and Icon are optional; adjust to your liking.
		Category: extutil.Ptr("AI"),
		Icon:     extutil.Ptr(""),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: clusterAttribute},
				{Attribute: namespaceAttribute},
				{Attribute: kindAttribute},
				{Attribute: nameAttribute},
				{Attribute: titleAttribute},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: clusterAttribute,
					Direction: "ASC",
				},
				{
					Attribute: namespaceAttribute,
					Direction: "ASC",
				},
				{
					Attribute: kindAttribute,
					Direction: "ASC",
				},
				{
					Attribute: nameAttribute,
					Direction: "ASC",
				},
				{
					Attribute: titleAttribute,
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *reliabilityIssueDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: "k8s.ai.reliability.key",
			Label: discovery_kit_api.PluralLabel{
				One:   "AI Reliability Key",
				Other: "AI Reliability Keys",
			},
		},
		{
			Attribute: "k8s.ai.reliability.issue.key",
			Label: discovery_kit_api.PluralLabel{
				One:   "AI Reliability Issue Key",
				Other: "AI Reliability Issue Keys",
			},
		},
		{
			Attribute: clusterAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "Kubernetes cluster (AI reliability)",
				Other: "Kubernetes clusters (AI reliability)",
			},
		},
		{
			Attribute: namespaceAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "Kubernetes namespace (AI reliability)",
				Other: "Kubernetes namespaces (AI reliability)",
			},
		},
		{
			Attribute: kindAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "Kubernetes kind (AI reliability)",
				Other: "Kubernetes kind (AI reliability)",
			},
		},
		{
			Attribute: nameAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "Kubernetes workload (AI reliability)",
				Other: "Kubernetes workloads (AI reliability)",
			},
		},
		{
			Attribute: titleAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "AI reliability issue title",
				Other: "AI reliability issue titles",
			},
		},
		{
			Attribute: "k8s.ai.reliability.category",
			Label: discovery_kit_api.PluralLabel{
				One:   "AI reliability issue category",
				Other: "AI reliability issue categories",
			},
		},
		{
			Attribute: "k8s.ai.reliability.severity",
			Label: discovery_kit_api.PluralLabel{
				One:   "AI reliability issue severity",
				Other: "AI reliability issue severities",
			},
		},
		{
			Attribute: "k8s.ai.reliability.priority",
			Label: discovery_kit_api.PluralLabel{
				One:   "AI reliability issue priority",
				Other: "AI reliability issue priorities",
			},
		},
		{
			Attribute: "k8s.ai.reliability.last-analysis",
			Label: discovery_kit_api.PluralLabel{
				One:   "Last AI reliability analysis",
				Other: "Last AI reliability analyses",
			},
		},
		{
			Attribute: "k8s.ai.reliability.raw",
			Label: discovery_kit_api.PluralLabel{
				One:   "Raw AI reliability JSON",
				Other: "Raw AI reliability JSONs",
			},
		},
	}
}

// DiscoverTargets reads all records from the in-memory store and exposes
// them as discovery targets.
func (d *reliabilityIssueDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	records := listSingleReliabilityIssueRecords()

	targets := make([]discovery_kit_api.Target, 0, len(records))

	for _, rec := range records {
		cluster, namespace, kind, name := splitKey(rec.WorkloadKey)
		if cluster == "" || namespace == "" || kind == "" || name == "" {
			// Skip malformed keys just in case.
			continue
		}

		id := rec.Key // workload key with issue index as unique ID

		attrs := map[string][]string{
			"k8s.ai.reliability.key":           {rec.WorkloadKey},
			"k8s.ai.reliability.issue.key":     {rec.Key},
			clusterAttribute:                   {cluster},
			namespaceAttribute:                 {namespace},
			kindAttribute:                      {kind},
			nameAttribute:                      {name},
			"k8s.ai.reliability.title":         {rec.Title},
			"k8s.ai.reliability.category":      {rec.Category},
			"k8s.ai.reliability.severity":      {rec.Severity},
			"k8s.ai.reliability.priority":      {rec.Priority},
			"k8s.ai.reliability.last-analysis": {rec.Timestamp.UTC().Format(time.RFC3339)},
			"k8s.ai.reliability.raw":           {rec.Raw},
			// Optionally mirror core k8s attributes to link this back to the
			// deployment targets in the UI:
			"k8s.cluster-name": {cluster},
			"k8s.namespace":    {namespace},
			"k8s.name":         {name},
		}

		targets = append(targets, discovery_kit_api.Target{
			Id:         id,
			TargetType: ReliabilityIssueTargetType,
			Label:      rec.Title,
			Attributes: attrs,
		})
	}

	return targets, nil
}

// listSingleReliabilityIssueRecords returns a snapshot of all issue records currently in memory.
// This uses the singleReliabilityIssuesStore defined in check_issues.go.
func listSingleReliabilityIssueRecords() []SingleReliabilityIssueRecord {
	singleReliabilityIssuesStore.mu.RLock()
	defer singleReliabilityIssuesStore.mu.RUnlock()

	result := make([]SingleReliabilityIssueRecord, 0, len(singleReliabilityIssuesStore.items))
	for _, rec := range singleReliabilityIssuesStore.items {
		result = append(result, rec)
	}
	return result
}

// splitKey expects "cluster/namespace/kind/name" and returns its parts.
func splitKey(key string) (cluster, namespace, kind, name string) {
	parts := strings.SplitN(key, "/", 4)
	if len(parts) != 4 {
		return "", "", "", ""
	}
	return parts[0], parts[1], parts[2], parts[3]
}
