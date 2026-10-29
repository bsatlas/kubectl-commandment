package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/kuberc"
)

func runKubercSet(_ context.Context, cfg *kubercSetConfig) error {
	path, _, err := kubercPath(cfg.Kuberc)
	if err != nil {
		return err
	}
	options := kuberc.NewSetOptions(cfg.Streams)
	options.KubeRCFile = path
	options.Section = cfg.Section
	options.Command = cfg.Command
	options.AliasName = cfg.Name
	options.Options = append([]string(nil), cfg.Option...)
	options.PrependArgs = append([]string(nil), cfg.Prependarg...)
	options.AppendArgs = append([]string(nil), cfg.Appendarg...)
	options.Overwrite = cfg.Overwrite
	options.PluginPolicy = cfg.Policy
	options.AllowlistEntries = append([]string(nil), cfg.AllowlistEntry...)
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}
