package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type configSetValueKind uint8

const (
	configSetValueUnknown configSetValueKind = iota
	configSetValueString
	configSetValueBool
	configSetValueBytes
)

type configSetAssignment struct {
	Property     configProperty
	Text         string
	Kind         configSetValueKind
	Bool         bool
	DecodedBytes []byte
	Base64Valid  bool
}

type configSetConfig struct {
	Kubeconfig  string
	Property    configProperty
	Value       string
	Assignment  configSetAssignment
	SetRawBytes bool
	Stdout      io.Writer
}

func parseConfigSetAssignment(property configProperty, text string) (configSetAssignment, error) {
	if text == "" {
		return configSetAssignment{}, cli.Rejection.Message(
			"kubeconfig-property-value",
			"you cannot use set to unset a property",
			cli.Rejection.At(0, 0, 0),
		)
	}

	kind, err := configSetPropertyValueKind(property)
	if err != nil {
		return configSetAssignment{}, cli.Rejection.Message(
			"kubeconfig-property",
			err.Error(),
			cli.Rejection.At(0, 0, len(property.Path)),
		)
	}

	assignment := configSetAssignment{
		Property:    property,
		Text:        text,
		Kind:        kind,
		Base64Valid: true,
	}
	switch kind {
	case configSetValueString:
	case configSetValueBool:
		assignment.Bool, err = strconv.ParseBool(text)
		if err != nil {
			return configSetAssignment{}, cli.Rejection.Message(
				"kubeconfig-property-value",
				err.Error(),
				cli.Rejection.At(0, 0, len(text)),
			)
		}
	case configSetValueBytes:
		decoded, decodeErr := base64.StdEncoding.DecodeString(text)
		assignment.Base64Valid = decodeErr == nil
		if decodeErr == nil {
			assignment.DecodedBytes = decoded
		}
	default:
		return configSetAssignment{}, errors.New("unsupported kubeconfig property value kind")
	}
	return assignment, nil
}

func configSetPropertyValueKind(property configProperty) (configSetValueKind, error) {
	switch property.Root {
	case configPropertyRootKind, configPropertyRootAPIVersion, configPropertyRootCurrentContext:
		return configSetValueString, nil
	case configPropertyRootPreferences:
		if property.HasField && property.Field == configPropertyFieldColors {
			return configSetValueBool, nil
		}
		return configSetValueUnknown, errors.New("can't set a map or structure to a value")
	case configPropertyRootClusters:
		if !property.HasName || !property.HasField {
			return configSetValueUnknown, errors.New("can't set a map to a value")
		}
		switch property.Field {
		case configPropertyFieldServer, configPropertyFieldTLSServerName, configPropertyFieldCertificateAuthority, configPropertyFieldProxyURL:
			return configSetValueString, nil
		case configPropertyFieldInsecureSkipTLSVerify, configPropertyFieldDisableCompression:
			return configSetValueBool, nil
		case configPropertyFieldCertificateAuthorityData:
			return configSetValueBytes, nil
		default:
			return configSetValueUnknown, unsupportedConfigProperty(property.Path)
		}
	case configPropertyRootUsers:
		if !property.HasName || !property.HasField {
			return configSetValueUnknown, errors.New("can't set a map to a value")
		}
		switch property.Field {
		case configPropertyFieldClientCertificate, configPropertyFieldClientKey, configPropertyFieldToken, configPropertyFieldTokenFile, configPropertyFieldActAs, configPropertyFieldActAsUID, configPropertyFieldUsername, configPropertyFieldPassword:
			return configSetValueString, nil
		case configPropertyFieldClientCertificateData, configPropertyFieldClientKeyData:
			return configSetValueBytes, nil
		case configPropertyFieldActAsGroups:
			return configSetValueUnknown, errors.New("unrecognized slice type. string")
		default:
			return configSetValueUnknown, unsupportedConfigProperty(property.Path)
		}
	case configPropertyRootContexts:
		if !property.HasName || !property.HasField {
			return configSetValueUnknown, errors.New("can't set a map to a value")
		}
		switch property.Field {
		case configPropertyFieldCluster, configPropertyFieldUser, configPropertyFieldNamespace:
			return configSetValueString, nil
		default:
			return configSetValueUnknown, unsupportedConfigProperty(property.Path)
		}
	case configPropertyRootExtensions:
		return configSetValueUnknown, errors.New("can't set a map to a value")
	case configPropertyRootUnknown:
		return configSetValueUnknown, unsupportedConfigProperty(property.Path)
	default:
		return configSetValueUnknown, unsupportedConfigProperty(property.Path)
	}
}

func configSetCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configSetConfig] {
	cmd := cli.Command[configSetConfig]{
		Name: "set",
		Description: cli.Description{
			Short: "Set an individual value in a kubeconfig file",
			Long:  "Set an individual value in a kubeconfig file.\n\n PROPERTY_NAME identifies a kubeconfig field or map entry. Dots separate structural fields; map keys themselves may contain dots.\n\n PROPERTY_VALUE is the new value you want to set. Binary fields such as 'certificate-authority-data' expect a base64 encoded string unless the --set-raw-bytes flag is used.\n\n Specifying an attribute name that already exists will merge new fields on top of existing values.",
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

			property := cb.Values.Value(&cb.Config.Property, configPropertyLanguage)
			cb.Arguments.ValueFor(property, "PROPERTY_NAME")

			value := cb.Values.String(&cb.Config.Value)
			cb.Arguments.ValueFor(value, "PROPERTY_VALUE")

			assignment := cb.Values.Derive(&cb.Config.Assignment, cli.Value.ComputedBy(func(_ context.Context, _ *configSetConfig, dependencies cli.Dependencies) (configSetAssignment, error) {
				propertyValue, ok := dependencies.Get(property)
				if !ok {
					return configSetAssignment{}, errors.New("kubeconfig property dependency is unavailable")
				}
				text, ok := dependencies.Get(value)
				if !ok {
					return configSetAssignment{}, errors.New("kubeconfig property value dependency is unavailable")
				}
				return parseConfigSetAssignment(propertyValue, text)
			}).From(property, value))

			setRawBytes := cb.Values.Bool(&cb.Config.SetRawBytes)
			setRawBytesFlag := cb.Flags.ValueFor(setRawBytes)
			cb.Values.ResolveWith(setRawBytesFlag, cli.Value.KeepLast())
			setRawBytesFlag.
				Description("When writing a []byte PROPERTY_VALUE, write the given string directly without base64 decoding.").
				Long("set-raw-bytes").
				ImplicitValue(true)
			cb.Rule.Check("kubectl-config-set-base64", func(_ context.Context, cfg *configSetConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Assignment.Kind != configSetValueBytes || cfg.SetRawBytes || cfg.Assignment.Base64Valid, nil
			}).
				Subject(assignment).
				DependsOn(setRawBytes.OptionalDependency())
		},
		Run: func(_ context.Context, cfg *configSetConfig) error {
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
		},
	}
	return cmd
}
