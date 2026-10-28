package main

import (
	"bytes"
	"fmt"

	"k8s.io/client-go/tools/clientcmd/api"
)

type configPropertyAction uint8

const (
	configPropertySet configPropertyAction = iota
	configPropertyUnset
)

type configPropertyMutation struct {
	Action configPropertyAction
	Text   string
	Bool   bool
	Bytes  []byte
}

func setConfigProperty(config *api.Config, assignment configSetAssignment, rawBytes bool) error {
	mutation := configPropertyMutation{
		Action: configPropertySet,
		Text:   assignment.Text,
		Bool:   assignment.Bool,
	}
	if assignment.Kind == configSetValueBytes {
		if rawBytes {
			mutation.Bytes = []byte(assignment.Text)
		} else {
			mutation.Bytes = bytes.Clone(assignment.DecodedBytes)
		}
	}
	return updateConfigProperty(config, assignment.Property, mutation)
}

func unsetConfigProperty(config *api.Config, property configProperty) error {
	mutation := configPropertyMutation{
		Action: configPropertyUnset,
	}
	return updateConfigProperty(config, property, mutation)
}

func updateConfigProperty(config *api.Config, property configProperty, mutation configPropertyMutation) error {
	unset := mutation.Action == configPropertyUnset
	switch property.Root {
	case configPropertyRootKind:
		return updateConfigString(&config.Kind, mutation)
	case configPropertyRootAPIVersion:
		return updateConfigString(&config.APIVersion, mutation)
	case configPropertyRootPreferences:
		return updatePreferencesProperty(&config.Preferences, property, mutation)
	case configPropertyRootClusters:
		return updateClusterProperty(config, property, mutation)
	case configPropertyRootUsers:
		return updateAuthInfoProperty(config, property, mutation)
	case configPropertyRootContexts:
		return updateContextProperty(config, property, mutation)
	case configPropertyRootCurrentContext:
		return updateConfigString(&config.CurrentContext, mutation)
	case configPropertyRootExtensions:
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		config.Extensions = nil
		return nil
	case configPropertyRootUnknown:
		return unsupportedConfigProperty(property.Path)
	default:
		return unsupportedConfigProperty(property.Path)
	}
}

func updatePreferencesProperty(preferences *api.Preferences, property configProperty, mutation configPropertyMutation) error {
	unset := mutation.Action == configPropertyUnset
	if !property.HasField {
		if !unset {
			return unsupportedConfigProperty(property.Path)
		}
		*preferences = api.Preferences{}
		return nil
	}

	switch property.Field {
	case configPropertyFieldColors:
		return updateConfigBool(&preferences.Colors, mutation)
	case configPropertyFieldExtensions:
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		preferences.Extensions = nil
		return nil
	default:
		return unsupportedConfigProperty(property.Path)
	}
}

func updateClusterProperty(config *api.Config, property configProperty, mutation configPropertyMutation) error {
	unset := mutation.Action == configPropertyUnset
	if !property.HasName {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		config.Clusters = nil
		return nil
	}
	if !property.HasField {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		if config.Clusters != nil {
			delete(config.Clusters, property.Name)
		}
		return nil
	}

	cluster := config.Clusters[property.Name]
	if cluster == nil {
		if unset {
			return invalidConfigMapKey(property.Name)
		}
		if config.Clusters == nil {
			config.Clusters = make(map[string]*api.Cluster)
		}
		cluster = &api.Cluster{}
		config.Clusters[property.Name] = cluster
	}

	switch property.Field {
	case configPropertyFieldServer:
		return updateConfigString(&cluster.Server, mutation)
	case configPropertyFieldTLSServerName:
		return updateConfigString(&cluster.TLSServerName, mutation)
	case configPropertyFieldInsecureSkipTLSVerify:
		return updateConfigBool(&cluster.InsecureSkipTLSVerify, mutation)
	case configPropertyFieldCertificateAuthority:
		return updateConfigString(&cluster.CertificateAuthority, mutation)
	case configPropertyFieldCertificateAuthorityData:
		return updateConfigBytes(&cluster.CertificateAuthorityData, mutation)
	case configPropertyFieldProxyURL:
		return updateConfigString(&cluster.ProxyURL, mutation)
	case configPropertyFieldDisableCompression:
		return updateConfigBool(&cluster.DisableCompression, mutation)
	case configPropertyFieldExtensions:
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		cluster.Extensions = nil
		return nil
	default:
		return unsupportedConfigProperty(property.Path)
	}
}

func updateAuthInfoProperty(config *api.Config, property configProperty, mutation configPropertyMutation) error {
	unset := mutation.Action == configPropertyUnset
	if !property.HasName {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		config.AuthInfos = nil
		return nil
	}
	if !property.HasField {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		if config.AuthInfos != nil {
			delete(config.AuthInfos, property.Name)
		}
		return nil
	}

	authInfo := config.AuthInfos[property.Name]
	if authInfo == nil {
		if unset {
			return invalidConfigMapKey(property.Name)
		}
		if config.AuthInfos == nil {
			config.AuthInfos = make(map[string]*api.AuthInfo)
		}
		authInfo = &api.AuthInfo{}
		config.AuthInfos[property.Name] = authInfo
	}

	switch property.Field {
	case configPropertyFieldClientCertificate:
		return updateConfigString(&authInfo.ClientCertificate, mutation)
	case configPropertyFieldClientCertificateData:
		return updateConfigBytes(&authInfo.ClientCertificateData, mutation)
	case configPropertyFieldClientKey:
		return updateConfigString(&authInfo.ClientKey, mutation)
	case configPropertyFieldClientKeyData:
		return updateConfigBytes(&authInfo.ClientKeyData, mutation)
	case configPropertyFieldToken:
		return updateConfigString(&authInfo.Token, mutation)
	case configPropertyFieldTokenFile:
		return updateConfigString(&authInfo.TokenFile, mutation)
	case configPropertyFieldActAs:
		return updateConfigString(&authInfo.Impersonate, mutation)
	case configPropertyFieldActAsUID:
		return updateConfigString(&authInfo.ImpersonateUID, mutation)
	case configPropertyFieldActAsGroups:
		if !unset {
			return fmt.Errorf("unrecognized slice type. string")
		}
		authInfo.ImpersonateGroups = nil
		return nil
	case configPropertyFieldActAsUserExtra:
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		authInfo.ImpersonateUserExtra = nil
		return nil
	case configPropertyFieldUsername:
		return updateConfigString(&authInfo.Username, mutation)
	case configPropertyFieldPassword:
		return updateConfigString(&authInfo.Password, mutation)
	case configPropertyFieldAuthProvider:
		if !unset {
			return unsupportedConfigProperty(property.Path)
		}
		authInfo.AuthProvider = nil
		return nil
	case configPropertyFieldExec:
		if !unset {
			return unsupportedConfigProperty(property.Path)
		}
		authInfo.Exec = nil
		return nil
	case configPropertyFieldExtensions:
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		authInfo.Extensions = nil
		return nil
	default:
		return unsupportedConfigProperty(property.Path)
	}
}

func updateContextProperty(config *api.Config, property configProperty, mutation configPropertyMutation) error {
	unset := mutation.Action == configPropertyUnset
	if !property.HasName {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		config.Contexts = nil
		return nil
	}
	if !property.HasField {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		if config.Contexts != nil {
			delete(config.Contexts, property.Name)
		}
		return nil
	}

	contextConfig := config.Contexts[property.Name]
	if contextConfig == nil {
		if unset {
			return invalidConfigMapKey(property.Name)
		}
		if config.Contexts == nil {
			config.Contexts = make(map[string]*api.Context)
		}
		contextConfig = &api.Context{}
		config.Contexts[property.Name] = contextConfig
	}

	switch property.Field {
	case configPropertyFieldCluster:
		return updateConfigString(&contextConfig.Cluster, mutation)
	case configPropertyFieldUser:
		return updateConfigString(&contextConfig.AuthInfo, mutation)
	case configPropertyFieldNamespace:
		return updateConfigString(&contextConfig.Namespace, mutation)
	case configPropertyFieldExtensions:
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		contextConfig.Extensions = nil
		return nil
	default:
		return unsupportedConfigProperty(property.Path)
	}
}

func updateConfigString(destination *string, mutation configPropertyMutation) error {
	if mutation.Action == configPropertyUnset {
		*destination = ""
		return nil
	}
	*destination = mutation.Text
	return nil
}

func updateConfigBool(destination *bool, mutation configPropertyMutation) error {
	if mutation.Action == configPropertyUnset {
		*destination = false
		return nil
	}
	*destination = mutation.Bool
	return nil
}

func updateConfigBytes(destination *[]byte, mutation configPropertyMutation) error {
	if mutation.Action == configPropertyUnset {
		*destination = nil
		return nil
	}
	*destination = bytes.Clone(mutation.Bytes)
	return nil
}

func invalidConfigMapKey(name string) error {
	return fmt.Errorf("current map key `%v` is invalid", name)
}
