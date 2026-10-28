package main

import (
	"context"
	"errors"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

func main() {
	plugins := discoverPluginCatalog()
	app, err := cli.Compile(applicationDefinition(plugins))
	if err == nil {
		err = app.RunFromProcess(context.Background())
	}
	if err != nil {
		if !errors.Is(err, errSilentExit) {
			_ = cli.WriteError(os.Stderr, err)
		}
		os.Exit(cli.ExitCode(err))
	}
}
