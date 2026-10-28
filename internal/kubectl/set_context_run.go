package kubectl

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func runConfigSetContext(_ context.Context, cfg *configSetContextConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}

	name := ""
	if len(cfg.Names) == 1 {
		name = cfg.Names[0]
	}
	if cfg.Current {
		if config.CurrentContext == "" {
			return errors.New("no current context is set")
		}
		name = config.CurrentContext
	}

	starting, exists := config.Contexts[name]
	if !exists {
		starting = api.NewContext()
	}
	modified := *starting
	if flagProvided(cfg.ClusterState) {
		modified.Cluster = cfg.Cluster
	}
	if flagProvided(cfg.UserState) {
		modified.AuthInfo = cfg.User
	}
	if flagProvided(cfg.NamespaceState) {
		modified.Namespace = cfg.Namespace
	}
	config.Contexts[name] = &modified

	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	if exists {
		_, err = fmt.Fprintf(cfg.Stdout, "Context %q modified.\n", name)
	} else {
		_, err = fmt.Fprintf(cfg.Stdout, "Context %q created.\n", name)
	}
	return err
}
