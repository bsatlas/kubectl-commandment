package main

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func autoscaleCreateOptions(fieldManager string, mode autoscaleDryRunMode) v1.CreateOptions {
	options := v1.CreateOptions{
		FieldManager: fieldManager,
	}
	if mode == autoscaleDryRunServer {
		options.DryRun = []string{v1.DryRunAll}
	}
	return options
}
