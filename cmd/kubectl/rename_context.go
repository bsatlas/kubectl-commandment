package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type configRenameContextConfig struct {
	Kubeconfig string
	Name       string
	NewName    string
	Stdout     io.Writer
}

func configRenameContextCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configRenameContextConfig] {
	cmd := cli.Command[configRenameContextConfig]{
		Name: "rename-context",
		Description: cli.Description{
			Short: "Rename a context from the kubeconfig file",
			Long:  "Renames a context from the kubeconfig file.\n\n CONTEXT_NAME is the context name that you want to change.\n\n NEW_NAME is the new name you want to set.\n\n Note: If the context being renamed is the 'current-context', this field will also be updated.",
		},
		Examples: []cli.Example{
			{
				Description: "Rename the context 'old-name' to 'new-name' in your kubeconfig file",
				CommandLine: "kubectl config rename-context old-name new-name",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configRenameContextConfig]) {
			kubeconfig := deriveKubeconfig(cb, &cb.Config.Kubeconfig, kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			name := cb.Values.String(&cb.Config.Name)
			newName := cb.Values.String(&cb.Config.NewName)
			cb.Arguments.ValueFor(name, "CONTEXT_NAME").
				CompleteBy(kubeconfigNameCompleter(
					func(cfg *configRenameContextConfig) string { return cfg.Kubeconfig },
					kubeconfigContextName,
				)).
				From(kubeconfig)
			cb.Arguments.ValueFor(newName, "NEW_NAME")
			cb.Rule.NotEmpty(newName)
		},
		Run: runConfigRenameContext,
	}
	return cmd
}
