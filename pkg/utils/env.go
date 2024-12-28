package utils

import (
	"os"
)

// GetEnv is a convenience wrapper for os.Getenv() with additional default value return
// when empty or unset
func GetEnv(envVar string, defaultValue string) string {
	val := os.Getenv(envVar)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetEnvBool is a convenience wrapper for os.Getenv() with additional default value return
func GetEnvBool(envVar string, defaultValue bool) bool {
	val := os.Getenv(envVar)
	if val == "" {
		return defaultValue
	}

	return val == "true"
}
