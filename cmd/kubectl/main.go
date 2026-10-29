package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	plugins := discoverPluginCatalog(os.Getenv("PATH"))
	app, err := application(plugins)
	if err == nil {
		err = app.RunFromProcess(context.Background())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
