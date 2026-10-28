package kubectl

import "k8s.io/kubectl/pkg/kuberc"

func kubercPath(path string) (string, bool, error) {
	return kuberc.LoadKuberc(path)
}
