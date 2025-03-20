// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extcommon

import (
	"context"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_statusPodCountCheckInternal(t *testing.T) {
	type preparedState struct {
		podCountCheckMode CheckMode
		initialCount      int
	}
	tests := []struct {
		name               string
		preparedState      preparedState
		readyCount         int
		desiredCount       int
		wantedErrorMessage *string
	}{
		{
			name: "podCountMin1Success",
			preparedState: preparedState{
				podCountCheckMode: PodCountMin1,
			},
			readyCount:         1,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountMin1Failure",
			preparedState: preparedState{
				podCountCheckMode: PodCountMin1,
			},
			readyCount:         0,
			wantedErrorMessage: extutil.Ptr("checkout has no ready pods."),
		},
		{
			name: "podCountEqualsDesiredCountSuccess",
			preparedState: preparedState{
				podCountCheckMode: PodCountEqualsDesiredCount,
			},
			readyCount:         2,
			desiredCount:       2,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountEqualsDesiredCountFailure",
			preparedState: preparedState{
				podCountCheckMode: PodCountEqualsDesiredCount,
			},
			readyCount:         1,
			desiredCount:       2,
			wantedErrorMessage: extutil.Ptr("checkout has 1 of desired 2 pods ready."),
		},
		{
			name: "podCountGreaterEqualsDesiredCountSuccess",
			preparedState: preparedState{
				podCountCheckMode: PodCountGreaterEqualsDesiredCount,
			},
			readyCount:         3,
			desiredCount:       2,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountGreaterEqualsDesiredCountFailure",
			preparedState: preparedState{
				podCountCheckMode: PodCountGreaterEqualsDesiredCount,
			},
			readyCount:         1,
			desiredCount:       2,
			wantedErrorMessage: extutil.Ptr("checkout has 1 of desired 2 pods ready."),
		},
		{
			name: "podCountLessThanDesiredCountSuccess",
			preparedState: preparedState{
				podCountCheckMode: PodCountLessThanDesiredCount,
			},
			readyCount:         1,
			desiredCount:       2,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountLessThanDesiredCountFailure",
			preparedState: preparedState{
				podCountCheckMode: PodCountLessThanDesiredCount,
			},
			readyCount:         2,
			desiredCount:       2,
			wantedErrorMessage: extutil.Ptr("checkout has all 2 desired pods ready."),
		},
		{
			name: "podCountIncreasedSuccess",
			preparedState: preparedState{
				podCountCheckMode: PodCountIncreased,
				initialCount:      1,
			},
			readyCount:         2,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountIncreasedFailure",
			preparedState: preparedState{
				podCountCheckMode: PodCountIncreased,
				initialCount:      2,
			},
			readyCount:         2,
			wantedErrorMessage: extutil.Ptr("checkout's pod count didn't increase. Initial count: 2, current count: 2."),
		},
		{
			name: "podCountDecreasedSuccess",
			preparedState: preparedState{
				podCountCheckMode: PodCountDecreased,
				initialCount:      2,
			},
			readyCount:         1,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountDecreasedFailure",
			preparedState: preparedState{
				podCountCheckMode: PodCountDecreased,
				initialCount:      2,
			},
			readyCount:         2,
			wantedErrorMessage: extutil.Ptr("checkout's pod count didn't decrease. Initial count: 2, current count: 2."),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			state := PodCountCheckState{
				Timeout:           time.Now().Add(time.Minute * -1),
				PodCountCheckMode: tt.preparedState.podCountCheckMode,
				Namespace:         "shop",
				Target:            "checkout",
				InitialCount:      tt.preparedState.initialCount,
			}

			action := &PodCountCheckAction{
				ActionId:   "test",
				TargetType: "testTargetType",
				GetDesiredAndCurrentPodCount: func(k8s *client.Client, namespace string, target string) (*int32, int32, error) {
					return extutil.Ptr(int32(tt.desiredCount)), int32(tt.readyCount), nil
				},
			}

			result, err := action.Status(context.Background(), &state)
			require.NoError(t, err)
			require.True(t, result.Completed)
			if tt.wantedErrorMessage != nil {
				assert.Equalf(t, *tt.wantedErrorMessage, result.Error.Title, "Error message should be %s", *tt.wantedErrorMessage)
			} else {
				assert.Nil(t, result.Error, "Error should be nil")
			}
		})
	}
}
