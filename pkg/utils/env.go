package utils

import (
	"os"
	"strconv"
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

// GetBoolEnv is a convenience wrapper for strconv.ParseBool()
func GetBoolEnv(envVar string) bool {
	val := os.Getenv(envVar)

	if val == "" {
		return false
	}

	if ret, err := strconv.ParseBool(val); err == nil {
		return ret
	}

	return false
}
