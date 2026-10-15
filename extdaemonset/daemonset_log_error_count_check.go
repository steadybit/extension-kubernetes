// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extdaemonset

import (
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDaemonSetLogErrorCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.LogErrorCountCheckState] {
	return extcommon.NewWorkloadLogErrorCountCheckAction(k8s, DaemonSetLogErrorCountCheckActionId, DaemonSetTargetType, "DaemonSet", "k8s.daemonset", false,
		func(k8s *client.Client, namespace, name string) metav1.Object {
			return k8s.DaemonSetByNamespaceAndName(namespace, name)
		})
}
