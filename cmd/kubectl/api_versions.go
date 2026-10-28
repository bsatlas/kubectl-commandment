package main

import (
	"context"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/apiresources"
)

type apiVersionsConfig struct {
	Client clientConfig
	Stdout io.Writer
}

func apiVersionsCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[apiVersionsConfig] {
	cmd := cli.Command[apiVersionsConfig]{
		Name: "api-versions",
		Description: cli.Description{
			Short: "Print the supported API versions on the server, in the form of \"group/version\"",
			Long:  "Print the supported API versions on the server, in the form of \"group/version\".",
		},
		Examples: []cli.Example{
			{
				Description: "Print the supported API versions",
				CommandLine: "kubectl api-versions",
			},
		},
		Declare: func(cb *cli.CommandBuilder[apiVersionsConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Stdout)
		},
		Run: runAPIVersions,
	}
	return cmd
}

func runAPIVersions(_ context.Context, cfg *apiVersionsConfig) error {
	flags := apiresources.NewAPIVersionsFlags(restClientGetter(cfg.Client), genericiooptions.IOStreams{
		Out: cfg.Stdout,
	})
	options, err := flags.ToOptions(nil)
	if err != nil {
		return err
	}
	return options.RunAPIVersions()
}
