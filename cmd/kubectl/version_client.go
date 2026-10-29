package main

import "k8s.io/component-base/version"

func kubectlClientVersion() (any, string) {
	info := version.Get()
	return &info, info.GitVersion
}
