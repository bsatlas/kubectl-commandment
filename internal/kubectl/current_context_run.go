package kubectl

import (
	"context"
	"fmt"
)

func runCurrentContext(_ context.Context, cfg *currentContextConfig) error {
	starting, err := configAccess(cfg.Kubeconfig).GetStartingConfig()
	if err != nil {
		return err
	}
	if starting.CurrentContext == "" {
		return fmt.Errorf("current-context is not set")
	}
	_, err = fmt.Fprintln(cfg.Stdout, starting.CurrentContext)
	return err
}
