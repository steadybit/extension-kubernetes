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
	clusterAttribute           = "k8s.ai.reliability_issues_issues.cluster"
	namespaceAttribute         = "k8s.ai.reliability_issues_issues.namespace"
	nameAttribute              = "k8s.ai.reliability_issues_issues.name"
	titleAttribute             = "k8s.ai.reliability_issues_issues.title"
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
		Label:   discovery_kit_api.PluralLabel{One: "Kubernetes issues", Other: "Kubernetes AI reliability issues"},
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		// Category and Icon are optional; adjust to your liking.
		Category: extutil.Ptr("AI"),
		Icon:     extutil.Ptr(targetIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: clusterAttribute},
				{Attribute: namespaceAttribute},
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
			Attribute: "k8s.ai.reliability_issues.key",
			Label: discovery_kit_api.PluralLabel{
				One:   "AI Local Store Key",
				Other: "AI Local Store Keys",
			},
		},
		{
			Attribute: clusterAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "cluster",
				Other: "clusters",
			},
		},
		{
			Attribute: namespaceAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "namespace",
				Other: "namespaces",
			},
		},
		{
			Attribute: nameAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "name",
				Other: "names",
			},
		},
		{
			Attribute: titleAttribute,
			Label: discovery_kit_api.PluralLabel{
				One:   "title",
				Other: "titles",
			},
		},
		{
			Attribute: "k8s.ai.reliability_issues.category",
			Label: discovery_kit_api.PluralLabel{
				One:   "category",
				Other: "categories",
			},
		},
		{
			Attribute: "k8s.ai.reliability_issues.severity",
			Label: discovery_kit_api.PluralLabel{
				One:   "severity",
				Other: "severities",
			},
		},
		{
			Attribute: "k8s.ai.reliability_issues.priority",
			Label: discovery_kit_api.PluralLabel{
				One:   "priority",
				Other: "priorities",
			},
		},
		{
			Attribute: "k8s.ai.reliability_issues.last-analysis",
			Label: discovery_kit_api.PluralLabel{
				One:   "last analysis",
				Other: "last analyses",
			},
		},
		{
			Attribute: "k8s.ai.reliability_issues.raw",
			Label: discovery_kit_api.PluralLabel{
				One:   "Raw JSON",
				Other: "Raw JSONs",
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
			"k8s.ai.reliability_issues.key":           {rec.WorkloadKey},
			"k8s.ai.reliability_issues.issue.key":     {rec.Key},
			clusterAttribute:                          {cluster},
			namespaceAttribute:                        {namespace},
			nameAttribute:                             {kind + "." + name},
			titleAttribute:                            {rec.Title},
			"k8s.ai.reliability_issues.category":      {rec.Category},
			"k8s.ai.reliability_issues.severity":      {rec.Severity},
			"k8s.ai.reliability_issues.priority":      {rec.Priority},
			"k8s.ai.reliability_issues.last-analysis": {rec.Timestamp.UTC().Format(time.RFC3339)},
			"k8s.ai.reliability_issues.raw":           {rec.Raw},
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
