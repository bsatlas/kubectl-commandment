package main

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/cli-runtime/pkg/resource"

	"k8s.io/kubectl/pkg/cmd/clusterinfo"
)

func runClusterInfo(_ context.Context, cfg *clusterInfoConfig) error {
	getter := restClientGetter(cfg.Client)
	restConfig, err := getter.ToRESTConfig()
	if err != nil {
		return err
	}
	namespace := cfg.Client.Namespace
	if namespace == "" {
		namespace = v1.NamespaceSystem
	}
	options := clusterinfo.ClusterInfoOptions{
		IOStreams: cfg.Streams,
		Namespace: namespace,
		Builder:   resource.NewBuilder(getter),
		Client:    restConfig,
	}
	return options.Run()
}
