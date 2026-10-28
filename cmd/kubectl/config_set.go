package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type configSetConfig struct {
	Kubeconfig    string
	PropertyName  string
	PropertyValue string
	SetRawBytes   bool
	Stdout        io.Writer
}

func configSetCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configSetConfig] {
	cmd := cli.Command[configSetConfig]{
		Name: "set",
		Description: cli.Description{
			Short: "Set an individual value in a kubeconfig file",
			Long:  "Set an individual value in a kubeconfig file.\n\n PROPERTY_NAME is a dot delimited name where each token represents either an attribute name or a map key.  Map keys may not contain dots.\n\n PROPERTY_VALUE is the new value you want to set. Binary fields such as 'certificate-authority-data' expect a base64 encoded string unless the --set-raw-bytes flag is used.\n\n Specifying an attribute name that already exists will merge new fields on top of existing values.",
		},
		Examples: []cli.Example{
			{
				Description: "Set the server field on the my-cluster cluster to https://1.2.3.4",
				CommandLine: "kubectl config set clusters.my-cluster.server https://1.2.3.4",
			},
			{
				Description: "Set the certificate-authority-data field on the my-cluster cluster",
				CommandLine: "kubectl config set clusters.my-cluster.certificate-authority-data $(echo \"cert_data_here\" | base64 -i -)",
			},
			{
				Description: "Set the cluster field in the my-context context to my-cluster",
				CommandLine: "kubectl config set contexts.my-context.cluster my-cluster",
			},
			{
				Description: "Set the client-key-data field in the cluster-admin user using --set-raw-bytes option",
				CommandLine: "kubectl config set users.cluster-admin.client-key-data cert_data_here --set-raw-bytes=true",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configSetConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			propertyName := cb.Values.String(&cb.Config.PropertyName)
			cb.Arguments.ValueFor(propertyName, "PROPERTY_NAME")
			propertyValue := cb.Values.String(&cb.Config.PropertyValue)
			cb.Arguments.ValueFor(propertyValue, "PROPERTY_VALUE")

			setRawBytes := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SetRawBytes))
			setRawBytes.Description("When writing a []byte PROPERTY_VALUE, write the given string directly without base64 decoding.").Long("set-raw-bytes").ImplicitValue(true)
		},
		Run: runConfigSet,
	}
	return cmd
}

func runConfigSet(_ context.Context, cfg *configSetConfig) error {
	if cfg.PropertyValue == "" {
		return errors.New("you cannot use set to unset a property")
	}
	if cfg.PropertyName == "" {
		return errors.New("you must specify a property")
	}

	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	if err := setConfigProperty(config, cfg.PropertyName, cfg.PropertyValue, cfg.SetRawBytes); err != nil {
		return err
	}
	if err := clientcmd.ModifyConfig(access, *config, false); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "Property %q set.\n", cfg.PropertyName)
	return err
}
