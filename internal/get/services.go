package get

import "k8s.io/cli-runtime/pkg/genericclioptions"

// Services are ordinary Kubernetes dependencies used by the get command.
type Services struct {
	RESTClientGetter func(ClientSelection) genericclioptions.RESTClientGetter
}

func (services Services) restClientGetter(selection ClientSelection) genericclioptions.RESTClientGetter {
	if services.RESTClientGetter != nil {
		return services.RESTClientGetter(selection)
	}
	flags := genericclioptions.NewConfigFlags(false)
	*flags.KubeConfig = selection.Kubeconfig
	*flags.Context = selection.Context
	*flags.Namespace = selection.Namespace
	return flags
}
