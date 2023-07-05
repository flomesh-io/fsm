package utils

import (
	"k8s.io/klog/v2"
	"strings"
)

func ParseEnabled(enabled string) bool {
	switch strings.ToLower(enabled) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off", "":
		return false
	}

	klog.Warningf("invalid syntax: %s, will be treated as false", enabled)

	return false
}
