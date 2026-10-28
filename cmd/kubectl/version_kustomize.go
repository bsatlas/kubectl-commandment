package main

import "k8s.io/kubectl/pkg/cmd/version"

func kubectlKustomizeVersion() string {
	if moduleVersion, ok := version.GetKustomizeModVersion(); ok {
		return moduleVersion
	}
	return "v5.8.3"
}
