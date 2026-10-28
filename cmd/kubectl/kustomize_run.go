package main

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kustomize/v5/commands/build"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func runKustomize(_ context.Context, cfg *kustomizeConfig) error {
	if len(cfg.Arguments) > 1 {
		return fmt.Errorf("specify one path to %s", konfig.DefaultKustomizationFileName())
	}
	path := filesys.SelfDir
	if len(cfg.Arguments) == 1 {
		path = cfg.Arguments[0]
	}

	options := krusty.MakeDefaultOptions()
	switch cfg.LoadRestrictor {
	case "", types.LoadRestrictionsRootOnly.String():
		options.LoadRestrictions = types.LoadRestrictionsRootOnly
	case "none", types.LoadRestrictionsNone.String():
		options.LoadRestrictions = types.LoadRestrictionsNone
	default:
		return fmt.Errorf("illegal flag value --load-restrictor %s; legal values: [%s %s]", cfg.LoadRestrictor, types.LoadRestrictionsRootOnly.String(), types.LoadRestrictionsNone.String())
	}
	switch cfg.Reorder {
	case "", string(krusty.ReorderOptionLegacy):
		options.Reorder = krusty.ReorderOptionLegacy
	case string(krusty.ReorderOptionNone):
		options.Reorder = krusty.ReorderOptionNone
	default:
		return fmt.Errorf("illegal flag value --reorder %s; legal values: [%s %s]", cfg.Reorder, krusty.ReorderOptionLegacy, krusty.ReorderOptionNone)
	}

	functionOptions := types.FnPluginLoadingOptions{
		Network:       cfg.Network,
		NetworkName:   cfg.NetworkName,
		Mounts:        append([]string(nil), cfg.Mount...),
		Env:           append([]string(nil), cfg.Env...),
		AsCurrentUser: cfg.AsCurrentUser,
	}
	if cfg.EnableAlphaPlugins {
		pluginConfig := types.EnabledPluginConfig(types.BploUseStaticallyLinked)
		pluginConfig.FnpLoadingOptions = functionOptions
		options.PluginConfig = pluginConfig
	} else {
		options.PluginConfig.HelmConfig.Enabled = cfg.EnableHelm
	}
	options.PluginConfig.HelmConfig.Command = cfg.HelmCommand
	options.PluginConfig.HelmConfig.ApiVersions = append([]string(nil), cfg.HelmAPIVersions...)
	options.PluginConfig.HelmConfig.KubeVersion = cfg.HelmKubeVersion
	options.PluginConfig.HelmConfig.Debug = cfg.HelmDebug
	options.AddManagedbyLabel = cfg.EnableManagedbyLabel || os.Getenv(konfig.EnableManagedbyLabelEnv) == "on"

	fileSystem := filesys.MakeFsOnDisk()
	result, err := krusty.MakeKustomizer(options).Run(fileSystem, path)
	if err != nil {
		return err
	}
	if cfg.Output != "" && fileSystem.IsDir(cfg.Output) {
		return build.MakeWriter(fileSystem).WriteIndividualFiles(cfg.Output, result)
	}
	yaml, err := result.AsYaml()
	if err != nil {
		return err
	}
	if cfg.Output != "" {
		return fileSystem.WriteFile(cfg.Output, yaml)
	}
	_, err = cfg.Stdout.Write(yaml)
	return err
}
