package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type configUnsetConfig struct {
	Kubeconfig   string
	PropertyName string
	Stdout       io.Writer
}

func configUnsetCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configUnsetConfig] {
	cmd := cli.Command[configUnsetConfig]{
		Name: "unset",
		Description: cli.Description{
			Short: "Unset an individual value in a kubeconfig file",
			Long:  "Unset an individual value in a kubeconfig file.\n\n PROPERTY_NAME is a dot delimited name where each token represents either an attribute name or a map key.  Map keys may not contain dots.",
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
			propertyName := cb.Values.String(&cb.Config.PropertyName)
			cb.Arguments.ValueFor(propertyName, "PROPERTY_NAME")
		},
		Run: runConfigUnset,
	}
	return cmd
}

func runConfigUnset(_ context.Context, cfg *configUnsetConfig) error {
	if cfg.PropertyName == "" {
		return errors.New("you must specify a property")
	}
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	if err := unsetConfigProperty(config, cfg.PropertyName); err != nil {
		return err
	}
	if err := clientcmd.ModifyConfig(access, *config, false); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "Property %q unset.\n", cfg.PropertyName)
	return err
}
