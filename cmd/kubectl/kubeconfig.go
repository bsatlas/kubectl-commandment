package main

import (
	"slices"

	"k8s.io/client-go/tools/clientcmd"
)

func configAccess(kubeconfig string) clientcmd.ConfigAccess {
	options := clientcmd.NewDefaultPathOptions()
	if kubeconfig != "" {
		options.LoadingRules.ExplicitPath = kubeconfig
	}
	return options
}

func configFile(access clientcmd.ConfigAccess) string {
	if access.IsExplicitFile() {
		return access.GetExplicitFile()
	}
	return access.GetDefaultFilename()
}

func clusterNames(kubeconfig string) ([]string, error) {
	config, err := configAccess(kubeconfig).GetStartingConfig()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(config.Clusters))
	for name := range config.Clusters {
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func contextNames(kubeconfig string) ([]string, error) {
	config, err := configAccess(kubeconfig).GetStartingConfig()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func userNames(kubeconfig string) ([]string, error) {
	config, err := configAccess(kubeconfig).GetStartingConfig()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(config.AuthInfos))
	for name := range config.AuthInfos {
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}
