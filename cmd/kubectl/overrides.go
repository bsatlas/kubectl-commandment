package main

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/cmd/util"
)

func applyOverrides(object, dataStruct runtime.Object, overrides, overrideType string) (runtime.Object, error) {
	options := util.OverrideOptions{
		Overrides:    overrides,
		OverrideType: util.OverrideType(overrideType),
	}
	return options.NewOverrider(dataStruct).Apply(object)
}
