package main

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestValidateCLIParams(t *testing.T) {
	testCases := []struct {
		name                       string
		meshName                   string
		fsmNamespace               string
		validatorWebhookConfigName string
		expectError                bool
	}{
		{
			name:                       "none of the necessary CLI params are empty",
			meshName:                   "test-mesh",
			fsmNamespace:               "test-ns",
			validatorWebhookConfigName: "test-webhook",
			expectError:                false,
		},
		{
			name:                       "mesh name is empty",
			meshName:                   "",
			fsmNamespace:               "test-ns",
			validatorWebhookConfigName: "test-webhook",
			expectError:                true,
		},
		{
			name:                       "fsm namespace is empty",
			meshName:                   "test-mesh",
			fsmNamespace:               "",
			validatorWebhookConfigName: "test-webhook",
			expectError:                true,
		},
		{
			name:                       "validator webhook is empty",
			meshName:                   "test-mesh",
			fsmNamespace:               "test-ns",
			validatorWebhookConfigName: "",
			expectError:                true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			meshName = tc.meshName
			fsmNamespace = tc.fsmNamespace
			validatorWebhookConfigName = tc.validatorWebhookConfigName
			err := validateCLIParams()
			assert.Equal(err != nil, tc.expectError)
		})
	}
}
