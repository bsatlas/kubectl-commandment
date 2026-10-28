package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"sigs.k8s.io/yaml"
)

type versionOutput struct {
	ClientVersion    any    `json:"clientVersion,omitempty" yaml:"clientVersion,omitempty"`
	KustomizeVersion string `json:"kustomizeVersion,omitempty" yaml:"kustomizeVersion,omitempty"`
	ServerVersion    any    `json:"serverVersion,omitempty" yaml:"serverVersion,omitempty"`
}

func runVersion(_ context.Context, cfg *versionConfig) error {
	if cfg.Output != "" && cfg.Output != "yaml" && cfg.Output != "json" {
		return errors.New("--output must be 'yaml' or 'json'")
	}

	client, clientGitVersion := kubectlClientVersion()
	output := versionOutput{
		ClientVersion:    client,
		KustomizeVersion: kubectlKustomizeVersion(),
	}

	var serverGitVersion string
	var serverErr error
	if !cfg.Client {
		discoveryClient, err := cfg.ClientConfig.factory().ToDiscoveryClient()
		if err != nil {
			serverErr = err
		} else {
			discoveryClient.Invalidate()
			server, err := discoveryClient.ServerVersion()
			if err != nil {
				serverErr = err
			} else {
				output.ServerVersion = server
				serverGitVersion = server.GitVersion
			}
		}
	}

	switch cfg.Output {
	case "":
		fmt.Fprintf(cfg.Streams.Out, "Client Version: %s\n", clientGitVersion)
		fmt.Fprintf(cfg.Streams.Out, "Kustomize Version: %s\n", output.KustomizeVersion)
		if serverGitVersion != "" {
			fmt.Fprintf(cfg.Streams.Out, "Server Version: %s\n", serverGitVersion)
		}
	case "yaml":
		marshalled, err := yaml.Marshal(&output)
		if err != nil {
			return err
		}
		fmt.Fprintln(cfg.Streams.Out, string(marshalled))
	case "json":
		marshalled, err := json.MarshalIndent(&output, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cfg.Streams.Out, string(marshalled))
	}

	if serverGitVersion != "" {
		warning, err := versionSkewWarning(clientGitVersion, serverGitVersion)
		if err != nil {
			return err
		}
		if warning != "" {
			if cfg.WarningsAsErrors {
				return errors.New(warning)
			}
			fmt.Fprintf(cfg.Streams.ErrOut, "Warning: %s\n", warning)
		}
	}
	return serverErr
}
