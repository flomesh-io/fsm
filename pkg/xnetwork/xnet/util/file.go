package util

import (
	"k8s.io/utils/mount"
)

// Exists checks whether the file exists
func Exists(name string) bool {
	if exists, err := mount.PathExists(name); err != nil {
		return false
	} else {
		return exists
	}
}
