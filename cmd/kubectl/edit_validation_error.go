package main

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func editInvalidError(err error) error {
	return errors.NewInvalid(
		v1.SchemeGroupVersion.WithKind("").GroupKind(),
		"",
		field.ErrorList{field.Invalid(nil, "The edited file failed validation", fmt.Sprintf("%v", err))},
	)
}
