package main

import (
	"context"
	"errors"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func compileSeparatorTestApplication[C any](
	t *testing.T,
	build func(cli.Destination[rootConfig, clientConfig]) cli.Command[C],
	run func(context.Context, *C) error,
) cli.Application {
	t.Helper()

	definition := cli.Definition[rootConfig]{
		Name:    "kubectl",
		Release: kubectlRelease,
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			client := root.Values.Derive(
				&root.Config.Client,
				cli.Value.ComputedBy(func(context.Context, *rootConfig, cli.Dependencies) (clientConfig, error) {
					return clientConfig{}, nil
				}),
			)
			command := build(client)
			command.Run = run
			root.Subcommands.Command(command)
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %+v cause=%v diagnostics=%#v", err, errors.Unwrap(err), cli.Diagnostics(err))
	}
	return app
}
