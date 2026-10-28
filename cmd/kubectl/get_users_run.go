package main

import (
	"context"
	"fmt"
)

func runConfigGetUsers(_ context.Context, cfg *configGetUsersConfig) error {
	names, err := userNames(cfg.Kubeconfig)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cfg.Stdout, "NAME"); err != nil {
		return err
	}
	for _, name := range names {
		if _, err := fmt.Fprintln(cfg.Stdout, name); err != nil {
			return err
		}
	}
	return nil
}
