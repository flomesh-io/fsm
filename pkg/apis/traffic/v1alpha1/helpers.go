package v1alpha1

import (
	"encoding/json"
	"errors"
)

// UnmarshalJSON converts a given array of single value maps to one map
func (h *httpHeaders) UnmarshalJSON(b []byte) error {
	*h = make(map[string]string)
	var temp []map[string]string
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	for _, m := range temp {
		if len(m) > 1 {
			return errors.New("incorrect length of keyval")
		}
		for key, val := range m {
			(*h)[key] = val
		}
	}
	return nil
}

// MarshalJSON converts a given map to array of single value maps
func (h httpHeaders) MarshalJSON() ([]byte, error) {
	var returnArr []map[string]string
	for key, val := range h {
		returnArr = append(returnArr, map[string]string{key: val})
	}
	return json.Marshal(returnArr)
}
