package main

import (
	"context"
	"fmt"

	"k8s.io/kubectl/pkg/cmd/create"
	"k8s.io/kubectl/pkg/cmd/util"
)

func runCreateToken(_ context.Context, cfg *createTokenConfig) error {
	factory := cfg.Client.factory()
	options := create.NewTokenOpts(cfg.Streams)
	options.Name = cfg.Name
	options.BoundObjectKind = cfg.BoundObjectKind
	options.BoundObjectName = cfg.BoundObjectName
	options.BoundObjectUID = cfg.BoundObjectUID
	options.Audiences = append([]string(nil), cfg.Audiences...)
	options.Duration = cfg.Duration

	var err error
	options.Namespace, _, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.CoreClient = clientset.CoreV1()
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, util.DryRunNone, cfg.Streams.Out)
	if err != nil {
		return err
	}
	if options.CoreClient == nil {
		return fmt.Errorf("no client provided")
	}
	if options.Namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	return options.Run()
}
