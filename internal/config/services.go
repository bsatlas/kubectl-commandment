package config

import "k8s.io/client-go/tools/clientcmd"

// Services are ordinary Kubernetes application dependencies used by config commands.
type Services struct {
	ConfigAccess func(string) clientcmd.ConfigAccess
}

func defaultConfigAccess(kubeconfig string) clientcmd.ConfigAccess {
	options := clientcmd.NewDefaultPathOptions()
	if kubeconfig != "" {
		options.LoadingRules.ExplicitPath = kubeconfig
	}
	return options
}

func (services Services) configAccess(kubeconfig string) clientcmd.ConfigAccess {
	if services.ConfigAccess != nil {
		return services.ConfigAccess(kubeconfig)
	}
	return defaultConfigAccess(kubeconfig)
}
