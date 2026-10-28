package main

import (
	"fmt"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

type configPropertyRoot uint8

const (
	configPropertyRootUnknown configPropertyRoot = iota
	configPropertyRootKind
	configPropertyRootAPIVersion
	configPropertyRootPreferences
	configPropertyRootClusters
	configPropertyRootUsers
	configPropertyRootContexts
	configPropertyRootCurrentContext
	configPropertyRootExtensions
)

type configPropertyField uint8

const (
	configPropertyFieldUnknown configPropertyField = iota
	configPropertyFieldColors
	configPropertyFieldExtensions
	configPropertyFieldServer
	configPropertyFieldTLSServerName
	configPropertyFieldInsecureSkipTLSVerify
	configPropertyFieldCertificateAuthority
	configPropertyFieldCertificateAuthorityData
	configPropertyFieldProxyURL
	configPropertyFieldDisableCompression
	configPropertyFieldClientCertificate
	configPropertyFieldClientCertificateData
	configPropertyFieldClientKey
	configPropertyFieldClientKeyData
	configPropertyFieldToken
	configPropertyFieldTokenFile
	configPropertyFieldActAs
	configPropertyFieldActAsUID
	configPropertyFieldActAsGroups
	configPropertyFieldActAsUserExtra
	configPropertyFieldUsername
	configPropertyFieldPassword
	configPropertyFieldAuthProvider
	configPropertyFieldExec
	configPropertyFieldCluster
	configPropertyFieldUser
	configPropertyFieldNamespace
)

type configProperty struct {
	Path     string
	Root     configPropertyRoot
	Name     string
	Field    configPropertyField
	HasName  bool
	HasField bool
}

func configPropertyLanguage() cli.ValueCodec[configProperty] {
	language := cli.ValueCodec[configProperty]{
		Name: "kubeconfig-property",
		Parse: func(text string) (configProperty, error) {
			property, err := parseConfigProperty(text)
			if err != nil {
				return configProperty{}, cli.Rejection.Message(
					"kubeconfig-property",
					err.Error(),
					cli.Rejection.At(0, 0, len(text)),
				)
			}
			return property, nil
		},
		Format:   func(property configProperty) string { return property.Path },
		Snapshot: func(property configProperty) configProperty { return property },
	}
	return language
}

func parseConfigProperty(path string) (configProperty, error) {
	parts := strings.Split(path, ".")
	if parts[0] == "" {
		return configProperty{}, fmt.Errorf("unable to parse one or more field values of %v", path)
	}

	property := configProperty{Path: path}
	switch parts[0] {
	case "kind":
		property.Root = configPropertyRootKind
		if len(parts) != 1 {
			return configProperty{}, unsupportedConfigProperty(path)
		}
	case "apiVersion":
		property.Root = configPropertyRootAPIVersion
		if len(parts) != 1 {
			return configProperty{}, unsupportedConfigProperty(path)
		}
	case "preferences":
		property.Root = configPropertyRootPreferences
		if err := parseConfigStructProperty(&property, parts[1:]); err != nil {
			return configProperty{}, err
		}
	case "clusters":
		property.Root = configPropertyRootClusters
		if err := parseConfigMapProperty(&property, parts[1:]); err != nil {
			return configProperty{}, err
		}
	case "users":
		property.Root = configPropertyRootUsers
		if err := parseConfigMapProperty(&property, parts[1:]); err != nil {
			return configProperty{}, err
		}
	case "contexts":
		property.Root = configPropertyRootContexts
		if err := parseConfigMapProperty(&property, parts[1:]); err != nil {
			return configProperty{}, err
		}
	case "current-context":
		property.Root = configPropertyRootCurrentContext
		if len(parts) != 1 {
			return configProperty{}, unsupportedConfigProperty(path)
		}
	case "extensions":
		property.Root = configPropertyRootExtensions
		if len(parts) != 1 {
			return configProperty{}, unsupportedConfigProperty(path)
		}
	default:
		return configProperty{}, fmt.Errorf("unable to parse %v at api.Config", path)
	}
	return property, nil
}

func parseConfigStructProperty(property *configProperty, parts []string) error {
	if len(parts) == 0 {
		return nil
	}
	if len(parts) != 1 {
		return unsupportedConfigProperty(property.Path)
	}
	field, ok := configPropertyFieldFor(property.Root, parts[0])
	if !ok {
		return unsupportedConfigProperty(property.Path)
	}
	property.Field = field
	property.HasField = true
	return nil
}

func parseConfigMapProperty(property *configProperty, parts []string) error {
	if len(parts) == 0 {
		return nil
	}
	property.HasName = true
	for index := 1; index < len(parts); index++ {
		field, ok := configPropertyFieldFor(property.Root, parts[index])
		if !ok {
			continue
		}
		if index+1 != len(parts) {
			return unsupportedConfigProperty(property.Path)
		}
		property.Name = strings.Join(parts[:index], ".")
		property.Field = field
		property.HasField = true
		return nil
	}
	property.Name = strings.Join(parts, ".")
	return nil
}

func configPropertyFieldFor(root configPropertyRoot, name string) (configPropertyField, bool) {
	field, ok := configPropertyFieldByName(name)
	if !ok || !configPropertyFieldAllowed(root, field) {
		return configPropertyFieldUnknown, false
	}
	return field, true
}

func configPropertyFieldByName(name string) (configPropertyField, bool) {
	switch name {
	case "colors":
		return configPropertyFieldColors, true
	case "extensions":
		return configPropertyFieldExtensions, true
	case "server":
		return configPropertyFieldServer, true
	case "tls-server-name":
		return configPropertyFieldTLSServerName, true
	case "insecure-skip-tls-verify":
		return configPropertyFieldInsecureSkipTLSVerify, true
	case "certificate-authority":
		return configPropertyFieldCertificateAuthority, true
	case "certificate-authority-data":
		return configPropertyFieldCertificateAuthorityData, true
	case "proxy-url":
		return configPropertyFieldProxyURL, true
	case "disable-compression":
		return configPropertyFieldDisableCompression, true
	case "client-certificate":
		return configPropertyFieldClientCertificate, true
	case "client-certificate-data":
		return configPropertyFieldClientCertificateData, true
	case "client-key":
		return configPropertyFieldClientKey, true
	case "client-key-data":
		return configPropertyFieldClientKeyData, true
	case "token":
		return configPropertyFieldToken, true
	case "tokenFile":
		return configPropertyFieldTokenFile, true
	case "act-as":
		return configPropertyFieldActAs, true
	case "act-as-uid":
		return configPropertyFieldActAsUID, true
	case "act-as-groups":
		return configPropertyFieldActAsGroups, true
	case "act-as-user-extra":
		return configPropertyFieldActAsUserExtra, true
	case "username":
		return configPropertyFieldUsername, true
	case "password":
		return configPropertyFieldPassword, true
	case "auth-provider":
		return configPropertyFieldAuthProvider, true
	case "exec":
		return configPropertyFieldExec, true
	case "cluster":
		return configPropertyFieldCluster, true
	case "user":
		return configPropertyFieldUser, true
	case "namespace":
		return configPropertyFieldNamespace, true
	default:
		return configPropertyFieldUnknown, false
	}
}

func configPropertyFieldAllowed(root configPropertyRoot, field configPropertyField) bool {
	switch root {
	case configPropertyRootPreferences:
		return field == configPropertyFieldColors || field == configPropertyFieldExtensions
	case configPropertyRootClusters:
		switch field {
		case configPropertyFieldServer,
			configPropertyFieldTLSServerName,
			configPropertyFieldInsecureSkipTLSVerify,
			configPropertyFieldCertificateAuthority,
			configPropertyFieldCertificateAuthorityData,
			configPropertyFieldProxyURL,
			configPropertyFieldDisableCompression,
			configPropertyFieldExtensions:
			return true
		default:
			return false
		}
	case configPropertyRootUsers:
		switch field {
		case configPropertyFieldClientCertificate,
			configPropertyFieldClientCertificateData,
			configPropertyFieldClientKey,
			configPropertyFieldClientKeyData,
			configPropertyFieldToken,
			configPropertyFieldTokenFile,
			configPropertyFieldActAs,
			configPropertyFieldActAsUID,
			configPropertyFieldActAsGroups,
			configPropertyFieldActAsUserExtra,
			configPropertyFieldUsername,
			configPropertyFieldPassword,
			configPropertyFieldAuthProvider,
			configPropertyFieldExec,
			configPropertyFieldExtensions:
			return true
		default:
			return false
		}
	case configPropertyRootContexts:
		switch field {
		case configPropertyFieldCluster,
			configPropertyFieldUser,
			configPropertyFieldNamespace,
			configPropertyFieldExtensions:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func unsupportedConfigProperty(path string) error {
	return fmt.Errorf("unable to parse one or more field values of %v", path)
}
