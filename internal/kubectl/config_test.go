package kubectl_test

import (
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func writeConfigTestKubeconfig(t *testing.T, current string, contexts ...string) string {
	t.Helper()

	config := api.NewConfig()
	config.CurrentContext = current
	config.Clusters["cluster"] = &api.Cluster{
		Server: "https://127.0.0.1",
	}
	config.AuthInfos["user"] = &api.AuthInfo{}
	for _, name := range contexts {
		config.Contexts[name] = &api.Context{
			Cluster:  "cluster",
			AuthInfo: "user",
		}
	}

	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}
	return path
}
