package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func main() {
	plugins := kubectl.DiscoverPluginCatalog(os.Getenv("PATH"))
	app, err := kubectl.Application(plugins)
	if err == nil {
		err = app.RunFromProcess(context.Background())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
