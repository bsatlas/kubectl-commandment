package main

import (
	"context"
	"os"

	"github.com/abstractionsinc/commandment/cli"
	kubectlcommandment "github.com/abstractionsinc/kubectl-commandment"
)

func main() {
	app, err := kubectlcommandment.Application()
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
	if err := app.RunFromProcess(context.Background()); err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
