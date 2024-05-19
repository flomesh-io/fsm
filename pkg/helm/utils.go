package helm

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadFile load a file from stdin, the local directory, or a remote file with a url.
func ReadFile(filePath string) ([]byte, error) {
	if strings.TrimSpace(filePath) == "-" {
		return io.ReadAll(os.Stdin)
	}

	return os.ReadFile(filepath.Clean(filePath))
}

// MergeMaps merges two maps together, with the values in the second map taking precedence.
func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
