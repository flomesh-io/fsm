package utils

import (
	"math/rand"
	"strings"
)

const (
	letters = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// ParseEnabled parses the given string to a boolean value.
func ParseEnabled(enabled string) bool {
	switch strings.ToLower(enabled) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off", "":
		return false
	}

	log.Warn().Msgf("invalid syntax: %s, will be treated as false", enabled)

	return false
}

func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}
