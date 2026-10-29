package main

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
)

func saveConfigAnnotation(enabled bool, object runtime.Object) error {
	return util.CreateOrUpdateAnnotation(enabled, object, scheme.DefaultJSONEncoder())
}
