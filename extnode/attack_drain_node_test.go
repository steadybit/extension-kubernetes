// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDrainNodeExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration": 100000,
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"host.hostname": {"test"},
			},
		}),
	}

	action := NewDrainNodeAction()
	state := action.NewEmptyState()

	// When
	_, err := action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Then
	require.Equal(t, "test", state.Node)
}

func TestExtractErrorFromStdOut(t *testing.T) {
	message := extractErrorFromStdOut(
		[]string{"abc\n",
			"error: unable to drain node \"ip-10-40-85-186.eu-central-1.compute.internal\" due to error:[error when evicting pods/\"istiod-969b65746-5rltp\" -n \"istio-system\": pods \"istiod-969b65746-5rltp\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"istio-system\", error when evicting pods/\"reviews-v3-58b6479b-d6cng\" -n \"istio-bookinfo\": pods \"reviews-v3-58b6479b-d6cng\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"istio-bookinfo\", error when evicting pods/\"orders-6f956dd8c6-dl54c\" -n \"steadybit-demo\": pods \"orders-6f956dd8c6-dl54c\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"gateway-8584c79587-tr8bk\" -n \"steadybit-demo\": pods \"gateway-8584c79587-tr8bk\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"checkout-8577c86785-srtg9\" -n \"steadybit-demo\": pods \"checkout-8577c86785-srtg9\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"fashion-bestseller-6758c976fd-gnzhp\" -n \"steadybit-demo\": pods \"fashion-bestseller-6758c976fd-gnzhp\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"productpage-v1-66756cddfd-4k49d\" -n \"istio-bookinfo\": pods \"productpage-v1-66756cddfd-4k49d\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"istio-bookinfo\", error when evicting pods/\"activemq-8457669cd7-g5smr\" -n \"steadybit-demo\": pods \"activemq-8457669cd7-g5smr\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"steadybit-extension-loadtest-58f94bcbd4-jp92v\" -n \"steadybit-outpost\": pods \"steadybit-extension-loadtest-58f94bcbd4-jp92v\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-outpost\", error when evicting pods/\"hot-deals-6dc558f6f8-bbghx\" -n \"steadybit-demo\": pods \"hot-deals-6dc558f6f8-bbghx\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"inventory-6f797fdcd7-sksrb\" -n \"steadybit-demo\": pods \"inventory -6f797fdcd7-sksrb\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"toys-bestseller-6799569767-wskrb\" -n \"steadybit-demo\": pods \"toys-bestseller-6799569767-wskrb\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\"], continuing command...",
			"def\n"})
	// Then
	require.Equal(t, "unable to drain node \"ip-10-40-85-186.eu-central-1.compute.internal\" due to error:[error when evicting pods/\"istiod-969b65746-5rltp\" -n \"istio-system\": pods \"istiod-969b65746-5rltp\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"istio-system\", error when evicting pods/\"reviews-v3-58b6479b-d6cng\" -n \"istio-bookinfo\": pods \"reviews-v3-58b6479b-d6cng\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"istio-bookinfo\", error when evicting pods/\"orders-6f956dd8c6-dl54c\" -n \"steadybit-demo\": pods \"orders-6f956dd8c6-dl54c\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"gateway-8584c79587-tr8bk\" -n \"steadybit-demo\": pods \"gateway-8584c79587-tr8bk\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"checkout-8577c86785-srtg9\" -n \"steadybit-demo\": pods \"checkout-8577c86785-srtg9\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"fashion-bestseller-6758c976fd-gnzhp\" -n \"steadybit-demo\": pods \"fashion-bestseller-6758c976fd-gnzhp\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"productpage-v1-66756cddfd-4k49d\" -n \"istio-bookinfo\": pods \"productpage-v1-66756cddfd-4k49d\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"istio-bookinfo\", error when evicting pods/\"activemq-8457669cd7-g5smr\" -n \"steadybit-demo\": pods \"activemq-8457669cd7-g5smr\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"steadybit-extension-loadtest-58f94bcbd4-jp92v\" -n \"steadybit-outpost\": pods \"steadybit-extension-loadtest-58f94bcbd4-jp92v\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-outpost\", error when evicting pods/\"hot-deals-6dc558f6f8-bbghx\" -n \"steadybit-demo\": pods \"hot-deals-6dc558f6f8-bbghx\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"inventory-6f797fdcd7-sksrb\" -n \"steadybit-demo\": pods \"inventory -6f797fdcd7-sksrb\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\", error when evicting pods/\"toys-bestseller-6799569767-wskrb\" -n \"steadybit-demo\": pods \"toys-bestseller-6799569767-wskrb\" is forbidden: User \"system:serviceaccount:steadybit-outpost:steadybit-extension-kubernetes\" cannot create resource \"pods/eviction\" in API group \"\" in the namespace \"steadybit-demo\"], continuing command...", *message)
}
