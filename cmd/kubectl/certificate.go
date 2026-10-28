package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type certificateConditionConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	Filename                 []string
	Force                    bool
	Kustomize                string
	Output                   string
	Recursive                bool
	ShowManagedFields        bool
	Template                 string
}

func certificateCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "certificate",
		Description: cli.Description{
			Short: "Modify certificate resources",
			Long:  "Modify certificate resources.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			approve := certificateApproveCommand(client)
			cb.Subcommands.Command(approve)
			deny := certificateDenyCommand(client)
			cb.Subcommands.Command(deny)
		},
	}
	return cmd
}
