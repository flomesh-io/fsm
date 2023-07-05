package utils

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"strings"
)

func ErrorListToError(errorList field.ErrorList) error {
	if errorList == nil {
		return nil
	}

	if len(errorList) == 0 {
		return nil
	}

	errorMsgs := sets.NewString()
	for _, err := range errorList {
		msg := fmt.Sprintf("%v", err)
		if errorMsgs.Has(msg) {
			continue
		}
		errorMsgs.Insert(msg)
	}

	return errors.New(strings.Join(errorMsgs.List(), ", "))
}
