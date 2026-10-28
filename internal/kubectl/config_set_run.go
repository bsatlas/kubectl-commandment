package kubectl

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

func runConfigSet(_ context.Context, cfg *configSetConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	if err := setConfigProperty(config, cfg.Assignment, cfg.SetRawBytes); err != nil {
		return err
	}
	if err := clientcmd.ModifyConfig(access, *config, false); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "Property %q set.\n", cfg.Assignment.Property.Path)
	return err
}
