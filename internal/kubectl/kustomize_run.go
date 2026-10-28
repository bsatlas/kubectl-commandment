package kubectl

import (
	"context"
	"slices"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kustomize/v5/commands/build"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func runKustomize(_ context.Context, cfg *kustomizeConfig) error {
	path := cfg.Directory
	if path == "" {
		path = filesys.SelfDir
	}

	options := krusty.MakeDefaultOptions()
	options.LoadRestrictions = cfg.LoadRestrictor
	if flagProvided(cfg.ReorderState) {
		options.Reorder = cfg.Reorder
	} else {
		options.Reorder = krusty.ReorderOptionUnspecified
	}

	functionOptions := types.FnPluginLoadingOptions{
		Network:       cfg.Network,
		NetworkName:   cfg.NetworkName,
		Mounts:        slices.Clone(cfg.Mount),
		Env:           slices.Clone(cfg.Env),
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
	options.PluginConfig.HelmConfig.ApiVersions = slices.Clone(cfg.HelmAPIVersions)
	options.PluginConfig.HelmConfig.KubeVersion = cfg.HelmKubeVersion
	options.PluginConfig.HelmConfig.Debug = cfg.HelmDebug
	options.AddManagedbyLabel = cfg.EnableManagedbyLabel || cfg.ManagedbyLabelEnv

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
