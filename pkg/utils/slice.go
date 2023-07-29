package utils

// StringsEqual compares two string slices and returns true if they are equal.
func StringsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := 0; i < len(left); i++ {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
