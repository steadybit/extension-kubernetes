// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extcommon

import (
	"context"
	"testing"
	"time"

	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			wantedErrorMessage: new("shop/checkout has no ready pods."),
		},
		{
			name: "podCountEquals0Success",
			preparedState: preparedState{
				podCountCheckMode: PodCountEquals0,
			},
			readyCount:         0,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountEquals0Failure",
			preparedState: preparedState{
				podCountCheckMode: PodCountEquals0,
			},
			readyCount:         2,
			wantedErrorMessage: new("shop/checkout has 2 ready pods, expected 0."),
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
			wantedErrorMessage: new("shop/checkout has 1 of desired 2 pods ready."),
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
			wantedErrorMessage: new("shop/checkout has 1 of desired 2 pods ready."),
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
			wantedErrorMessage: new("shop/checkout has all 2 desired pods ready."),
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
			wantedErrorMessage: new("shop/checkout's pod count didn't increase. Initial count: 2, current count: 2."),
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
			wantedErrorMessage: new("shop/checkout's pod count didn't decrease. Initial count: 2, current count: 2."),
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
					return new(int32(tt.desiredCount)), int32(tt.readyCount), nil
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

func Test_statusPodCountCheckStatusCheckMode(t *testing.T) {
	tests := []struct {
		name            string
		statusCheckMode StatusCheckMode
		failEarly       bool
		timeoutElapsed  bool
		readyCount      int
		desiredCount    int
		wantCompleted   bool
		wantError       bool
	}{
		{
			name:            "atLeastOnce succeeds immediately when condition is met",
			statusCheckMode: StatusCheckModeAtLeastOnce,
			timeoutElapsed:  false,
			readyCount:      2,
			desiredCount:    2,
			wantCompleted:   true,
			wantError:       false,
		},
		{
			name:            "atLeastOnce keeps running while condition is not met and timeout not reached",
			statusCheckMode: StatusCheckModeAtLeastOnce,
			timeoutElapsed:  false,
			readyCount:      1,
			desiredCount:    2,
			wantCompleted:   false,
			wantError:       false,
		},
		{
			name:            "atLeastOnce fails once timeout reached and condition never met",
			statusCheckMode: StatusCheckModeAtLeastOnce,
			timeoutElapsed:  true,
			readyCount:      1,
			desiredCount:    2,
			wantCompleted:   true,
			wantError:       true,
		},
		{
			name:            "empty status check mode defaults to atLeastOnce",
			statusCheckMode: "",
			timeoutElapsed:  false,
			readyCount:      2,
			desiredCount:    2,
			wantCompleted:   true,
			wantError:       false,
		},
		{
			name:            "allTheTime keeps running while condition holds and timeout not reached",
			statusCheckMode: StatusCheckModeAllTheTime,
			timeoutElapsed:  false,
			readyCount:      2,
			desiredCount:    2,
			wantCompleted:   false,
			wantError:       false,
		},
		{
			name:            "allTheTime with fail early fails fast when condition is violated",
			statusCheckMode: StatusCheckModeAllTheTime,
			failEarly:       true,
			timeoutElapsed:  false,
			readyCount:      1,
			desiredCount:    2,
			wantCompleted:   true,
			wantError:       true,
		},
		{
			name:            "allTheTime without fail early keeps running on violation before timeout",
			statusCheckMode: StatusCheckModeAllTheTime,
			failEarly:       false,
			timeoutElapsed:  false,
			readyCount:      1,
			desiredCount:    2,
			wantCompleted:   false,
			wantError:       false,
		},
		{
			name:            "allTheTime without fail early reports the violation once the duration has elapsed",
			statusCheckMode: StatusCheckModeAllTheTime,
			failEarly:       false,
			timeoutElapsed:  true,
			readyCount:      1,
			desiredCount:    2,
			wantCompleted:   true,
			wantError:       true,
		},
		{
			name:            "allTheTime succeeds once the duration has elapsed without violation",
			statusCheckMode: StatusCheckModeAllTheTime,
			timeoutElapsed:  true,
			readyCount:      2,
			desiredCount:    2,
			wantCompleted:   true,
			wantError:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := time.Now().Add(time.Minute)
			if tt.timeoutElapsed {
				timeout = time.Now().Add(-time.Minute)
			}

			state := PodCountCheckState{
				Timeout:           timeout,
				PodCountCheckMode: PodCountEqualsDesiredCount,
				StatusCheckMode:   tt.statusCheckMode,
				FailEarly:         tt.failEarly,
				Namespace:         "shop",
				Target:            "checkout",
			}

			action := &PodCountCheckAction{
				ActionId:   "test",
				TargetType: "testTargetType",
				GetDesiredAndCurrentPodCount: func(k8s *client.Client, namespace string, target string) (*int32, int32, error) {
					return new(int32(tt.desiredCount)), int32(tt.readyCount), nil
				},
			}

			result, err := action.Status(context.Background(), &state)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCompleted, result.Completed)
			if tt.wantError {
				assert.NotNil(t, result.Error)
			} else {
				assert.Nil(t, result.Error)
			}
		})
	}
}
