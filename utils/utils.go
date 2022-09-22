// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package utils

import (
	"github.com/mitchellh/mapstructure"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/exthttp"
	"net/http"
)

func WriteActionState[T any](w http.ResponseWriter, state T) {
	err, encodedState := EncodeActionState(state)
	if err != nil {
		exthttp.WriteError(w, extension_kit.ToError("Failed to encode attack state", err))
	} else {
		exthttp.WriteBody(w, action_kit_api.PrepareResult{
			State: encodedState,
		})
	}
}

func EncodeActionState[T any](attackState T) (error, action_kit_api.ActionState) {
	var result action_kit_api.ActionState
	err := mapstructure.Decode(attackState, &result)
	return err, result
}

func DecodeActionState[T any](attackState action_kit_api.ActionState, result *T) error {
	return mapstructure.Decode(attackState, result)
}
