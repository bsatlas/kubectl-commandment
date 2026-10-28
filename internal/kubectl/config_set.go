package kubectl

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
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
	Assignment  configSetAssignment
	SetRawBytes bool
	Stdout      io.Writer
}

func snapshotConfigSetAssignment(assignment configSetAssignment) configSetAssignment {
	assignment.DecodedBytes = bytes.Clone(assignment.DecodedBytes)
	return assignment
}

func configSetAssignmentLanguage() cli.TokenLanguage[configSetAssignment] {
	language := cli.TokenLanguage[configSetAssignment]{
		Name:   "kubectl-config-set-assignment",
		Count:  cli.Count.Exactly(2),
		Syntax: []string{"PROPERTY_NAME", "PROPERTY_VALUE"},
		Parse: func(_ context.Context, tokens []string) (configSetAssignment, error) {
			property, err := parseConfigProperty(tokens[0])
			if err != nil {
				return configSetAssignment{}, cli.Rejection.Message(
					"kubeconfig-property",
					err.Error(),
					cli.Rejection.At(0, 0, len(tokens[0])),
				)
			}
			if tokens[1] == "" {
				return configSetAssignment{}, cli.Rejection.Message(
					"kubeconfig-property-value",
					"you cannot use set to unset a property",
					cli.Rejection.At(1, 0, 0),
				)
			}

			kind, err := configSetPropertyValueKind(property)
			if err != nil {
				return configSetAssignment{}, cli.Rejection.Message(
					"kubeconfig-property",
					err.Error(),
					cli.Rejection.At(0, 0, len(tokens[0])),
				)
			}

			assignment := configSetAssignment{
				Property:    property,
				Text:        tokens[1],
				Kind:        kind,
				Base64Valid: true,
			}
			switch kind {
			case configSetValueString:
			case configSetValueBool:
				assignment.Bool, err = strconv.ParseBool(tokens[1])
				if err != nil {
					return configSetAssignment{}, cli.Rejection.Message(
						"kubeconfig-property-value",
						err.Error(),
						cli.Rejection.At(1, 0, len(tokens[1])),
					)
				}
			case configSetValueBytes:
				decoded, decodeErr := base64.StdEncoding.DecodeString(tokens[1])
				assignment.Base64Valid = decodeErr == nil
				if decodeErr == nil {
					assignment.DecodedBytes = decoded
				}
			default:
				return configSetAssignment{}, errors.New("unsupported kubeconfig property value kind")
			}
			return assignment, nil
		},
		Format: func(assignment configSetAssignment) []string {
			return []string{assignment.Property.Path, assignment.Text}
		},
	}
	return language
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

			assignment := cb.Values.Destination(&cb.Config.Assignment, snapshotConfigSetAssignment)
			cb.Arguments.Tokens(assignment, "PROPERTY", configSetAssignmentLanguage())

			setRawBytes := cb.Values.Bool(&cb.Config.SetRawBytes)
			cb.Flags.ValueFor(setRawBytes).
				Description("When writing a []byte PROPERTY_VALUE, write the given string directly without base64 decoding.").
				Long("set-raw-bytes").
				ImplicitValue(true)
			cb.Rule.Check("kubectl-config-set-base64", func(_ context.Context, cfg *configSetConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Assignment.Kind != configSetValueBytes || cfg.SetRawBytes || cfg.Assignment.Base64Valid, nil
			}).
				Subject(assignment).
				DependsOn(setRawBytes.OptionalDependency())
		},
		Run: runConfigSet,
	}
	return cmd
}
