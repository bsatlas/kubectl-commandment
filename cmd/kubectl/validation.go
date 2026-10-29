package main

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func validationDirective(value string) (string, error) {
	boolean, err := strconv.ParseBool(value)
	if err == nil {
		if boolean {
			return v1.FieldValidationStrict, nil
		}
		return v1.FieldValidationIgnore, nil
	}

	switch value {
	case "strict", "":
		return v1.FieldValidationStrict, nil
	case "warn":
		return v1.FieldValidationWarn, nil
	case "ignore":
		return v1.FieldValidationIgnore, nil
	default:
		return v1.FieldValidationStrict, fmt.Errorf("invalid validate option %q; must be strict, warn, or ignore", value)
	}
}
