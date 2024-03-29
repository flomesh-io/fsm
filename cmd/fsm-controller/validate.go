package main

import (
	"fmt"
)

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if meshName == "" {
		return fmt.Errorf("Please specify the mesh name using --mesh-name")
	}

	if fsmNamespace == "" {
		return fmt.Errorf("Please specify the FSM namespace using --fsm-namespace")
	}

	if validatorWebhookConfigName == "" {
		return fmt.Errorf("Please specify the webhook configuration name using --validator-webhook-config")
	}

	return nil
}
