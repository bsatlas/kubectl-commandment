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
			mutation.Bytes = []byte(assignment.DecodedBytes)
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
	case "colors":
		return updateConfigBool(&preferences.Colors, mutation)
	case "extensions":
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
	case "server":
		return updateConfigString(&cluster.Server, mutation)
	case "tls-server-name":
		return updateConfigString(&cluster.TLSServerName, mutation)
	case "insecure-skip-tls-verify":
		return updateConfigBool(&cluster.InsecureSkipTLSVerify, mutation)
	case "certificate-authority":
		return updateConfigString(&cluster.CertificateAuthority, mutation)
	case "certificate-authority-data":
		return updateConfigBytes(&cluster.CertificateAuthorityData, mutation)
	case "proxy-url":
		return updateConfigString(&cluster.ProxyURL, mutation)
	case "disable-compression":
		return updateConfigBool(&cluster.DisableCompression, mutation)
	case "extensions":
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
	case "client-certificate":
		return updateConfigString(&authInfo.ClientCertificate, mutation)
	case "client-certificate-data":
		return updateConfigBytes(&authInfo.ClientCertificateData, mutation)
	case "client-key":
		return updateConfigString(&authInfo.ClientKey, mutation)
	case "client-key-data":
		return updateConfigBytes(&authInfo.ClientKeyData, mutation)
	case "token":
		return updateConfigString(&authInfo.Token, mutation)
	case "tokenFile":
		return updateConfigString(&authInfo.TokenFile, mutation)
	case "act-as":
		return updateConfigString(&authInfo.Impersonate, mutation)
	case "act-as-uid":
		return updateConfigString(&authInfo.ImpersonateUID, mutation)
	case "act-as-groups":
		if !unset {
			return fmt.Errorf("unrecognized slice type. string")
		}
		authInfo.ImpersonateGroups = nil
		return nil
	case "act-as-user-extra":
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		authInfo.ImpersonateUserExtra = nil
		return nil
	case "username":
		return updateConfigString(&authInfo.Username, mutation)
	case "password":
		return updateConfigString(&authInfo.Password, mutation)
	case "auth-provider":
		if !unset {
			return unsupportedConfigProperty(property.Path)
		}
		authInfo.AuthProvider = nil
		return nil
	case "exec":
		if !unset {
			return unsupportedConfigProperty(property.Path)
		}
		authInfo.Exec = nil
		return nil
	case "extensions":
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
	case "cluster":
		return updateConfigString(&contextConfig.Cluster, mutation)
	case "user":
		return updateConfigString(&contextConfig.AuthInfo, mutation)
	case "namespace":
		return updateConfigString(&contextConfig.Namespace, mutation)
	case "extensions":
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

func unsupportedConfigProperty(path string) error {
	return fmt.Errorf("unable to parse one or more field values of %v", path)
}

func invalidConfigMapKey(name string) error {
	return fmt.Errorf("current map key `%v` is invalid", name)
}
