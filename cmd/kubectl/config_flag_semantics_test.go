package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestConfigSetClusterTristateFlagPreservesBareAndExplicitFalse(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config")
	starting := api.NewConfig()
	starting.Clusters["e2e"] = &api.Cluster{
		InsecureSkipTLSVerify: true,
	}
	if err := clientcmd.WriteToFile(*starting, path); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	definition := cli.Definition[rootConfig]{
		Name: "kubectl",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			kubeconfig := root.Values.String(&root.Config.Kubeconfig)
			root.Values.Default(kubeconfig, path)
			root.Subcommands.Command(configSetClusterCommand(kubeconfig))
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var stdout bytes.Buffer
	err = app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"set-cluster", "e2e", "--insecure-skip-tls-verify=false"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run(false) error = %v", err)
	}
	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile(false) error = %v", err)
	}
	if config.Clusters["e2e"].InsecureSkipTLSVerify {
		t.Fatal("insecure-skip-tls-verify after explicit false = true, want false")
	}

	stdout.Reset()
	err = app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"set-cluster", "e2e", "--insecure-skip-tls-verify"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run(bare) error = %v", err)
	}
	config, err = clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile(bare) error = %v", err)
	}
	if !config.Clusters["e2e"].InsecureSkipTLSVerify {
		t.Fatal("insecure-skip-tls-verify after bare flag = false, want true")
	}
}
