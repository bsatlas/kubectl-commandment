package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type configUnsetConfig struct {
	Kubeconfig string
	Property   configProperty
	Stdout     io.Writer
}

func configUnsetCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configUnsetConfig] {
	cmd := cli.Command[configUnsetConfig]{
		Name: "unset",
		Description: cli.Description{
			Short: "Unset an individual value in a kubeconfig file",
			Long:  "Unset an individual value in a kubeconfig file.\n\n PROPERTY_NAME identifies a kubeconfig field or map entry. Dots separate structural fields; map keys themselves may contain dots.",
		},
		Examples: []cli.Example{
			{
				Description: "Unset the current-context",
				CommandLine: "kubectl config unset current-context",
			},
			{
				Description: "Unset namespace in foo context",
				CommandLine: "kubectl config unset contexts.foo.namespace",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configUnsetConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)
			property := cb.Values.Value(&cb.Config.Property, configPropertyLanguage)
			cb.Arguments.ValueFor(property, "PROPERTY_NAME")
		},
		Run: func(_ context.Context, cfg *configUnsetConfig) error {
			access := configAccess(cfg.Kubeconfig)
			config, err := access.GetStartingConfig()
			if err != nil {
				return err
			}
			if err := unsetConfigProperty(config, cfg.Property); err != nil {
				return err
			}
			if err := clientcmd.ModifyConfig(access, *config, false); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cfg.Stdout, "Property %q unset.\n", cfg.Property.Path)
			return err
		},
	}
	return cmd
}
