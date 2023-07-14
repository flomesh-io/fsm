package utils

import "time"

// Hash calculates an FNV-1 hash from a given byte array
func Hash(bytes []byte) uint64 {
	if hashCode, err := HashFromString(string(bytes)); err == nil {
		return hashCode
	}
	return uint64(time.Now().Nanosecond())
}
