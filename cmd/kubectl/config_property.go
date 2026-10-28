package main

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/client-go/tools/clientcmd/api"
)

type configPropertyAction uint8

const (
	configPropertySet configPropertyAction = iota
	configPropertyUnset
)

func setConfigProperty(config *api.Config, path, propertyValue string, rawBytes bool) error {
	return updateConfigProperty(config, path, propertyValue, configPropertySet, rawBytes)
}

func unsetConfigProperty(config *api.Config, path string) error {
	return updateConfigProperty(config, path, "", configPropertyUnset, false)
}

var clusterConfigFields = map[string]struct{}{
	"server":                     {},
	"tls-server-name":            {},
	"insecure-skip-tls-verify":   {},
	"certificate-authority":      {},
	"certificate-authority-data": {},
	"proxy-url":                  {},
	"disable-compression":        {},
	"extensions":                 {},
}

var authInfoConfigFields = map[string]struct{}{
	"client-certificate":      {},
	"client-certificate-data": {},
	"client-key":              {},
	"client-key-data":         {},
	"token":                   {},
	"tokenFile":               {},
	"act-as":                  {},
	"act-as-uid":              {},
	"act-as-groups":           {},
	"act-as-user-extra":       {},
	"username":                {},
	"password":                {},
	"auth-provider":           {},
	"exec":                    {},
	"extensions":              {},
}

var contextConfigFields = map[string]struct{}{
	"cluster":    {},
	"user":       {},
	"namespace":  {},
	"extensions": {},
}

func updateConfigProperty(config *api.Config, path, propertyValue string, action configPropertyAction, rawBytes bool) error {
	unset := action == configPropertyUnset
	parts := strings.Split(path, ".")
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("unable to parse one or more field values of %v", path)
	}

	switch parts[0] {
	case "kind":
		return updateConfigString(&config.Kind, path, parts[1:], propertyValue, action)
	case "apiVersion":
		return updateConfigString(&config.APIVersion, path, parts[1:], propertyValue, action)
	case "preferences":
		return updatePreferencesProperty(&config.Preferences, path, parts[1:], propertyValue, action)
	case "clusters":
		return updateClusterProperty(config, path, parts[1:], propertyValue, action, rawBytes)
	case "users":
		return updateAuthInfoProperty(config, path, parts[1:], propertyValue, action, rawBytes)
	case "contexts":
		return updateContextProperty(config, path, parts[1:], propertyValue, action)
	case "current-context":
		return updateConfigString(&config.CurrentContext, path, parts[1:], propertyValue, action)
	case "extensions":
		if len(parts) != 1 {
			return unsupportedConfigProperty(path)
		}
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		config.Extensions = nil
		return nil
	default:
		return fmt.Errorf("unable to parse %v at api.Config", path)
	}
}

func updatePreferencesProperty(preferences *api.Preferences, path string, parts []string, propertyValue string, action configPropertyAction) error {
	unset := action == configPropertyUnset
	if len(parts) == 0 {
		if !unset {
			return unsupportedConfigProperty(path)
		}
		*preferences = api.Preferences{}
		return nil
	}
	if len(parts) != 1 {
		return unsupportedConfigProperty(path)
	}

	switch parts[0] {
	case "colors":
		return updateConfigBool(&preferences.Colors, propertyValue, action)
	case "extensions":
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		preferences.Extensions = nil
		return nil
	default:
		return unsupportedConfigProperty(path)
	}
}

func updateClusterProperty(config *api.Config, path string, parts []string, propertyValue string, action configPropertyAction, rawBytes bool) error {
	unset := action == configPropertyUnset
	if len(parts) == 0 {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		config.Clusters = nil
		return nil
	}

	name, fields := splitConfigMapProperty(parts, clusterConfigFields)
	if len(fields) == 0 {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		if config.Clusters != nil {
			delete(config.Clusters, name)
		}
		return nil
	}
	if len(fields) != 1 {
		return unsupportedConfigProperty(path)
	}

	cluster := config.Clusters[name]
	if cluster == nil {
		if unset {
			return invalidConfigMapKey(name)
		}
		if config.Clusters == nil {
			config.Clusters = make(map[string]*api.Cluster)
		}
		cluster = &api.Cluster{}
		config.Clusters[name] = cluster
	}

	switch fields[0] {
	case "server":
		return updateConfigString(&cluster.Server, path, nil, propertyValue, action)
	case "tls-server-name":
		return updateConfigString(&cluster.TLSServerName, path, nil, propertyValue, action)
	case "insecure-skip-tls-verify":
		return updateConfigBool(&cluster.InsecureSkipTLSVerify, propertyValue, action)
	case "certificate-authority":
		return updateConfigString(&cluster.CertificateAuthority, path, nil, propertyValue, action)
	case "certificate-authority-data":
		return updateConfigBytes(&cluster.CertificateAuthorityData, propertyValue, action, rawBytes)
	case "proxy-url":
		return updateConfigString(&cluster.ProxyURL, path, nil, propertyValue, action)
	case "disable-compression":
		return updateConfigBool(&cluster.DisableCompression, propertyValue, action)
	case "extensions":
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		cluster.Extensions = nil
		return nil
	default:
		return unsupportedConfigProperty(path)
	}
}

func updateAuthInfoProperty(config *api.Config, path string, parts []string, propertyValue string, action configPropertyAction, rawBytes bool) error {
	unset := action == configPropertyUnset
	if len(parts) == 0 {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		config.AuthInfos = nil
		return nil
	}

	name, fields := splitConfigMapProperty(parts, authInfoConfigFields)
	if len(fields) == 0 {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		if config.AuthInfos != nil {
			delete(config.AuthInfos, name)
		}
		return nil
	}
	if len(fields) != 1 {
		return unsupportedConfigProperty(path)
	}

	authInfo := config.AuthInfos[name]
	if authInfo == nil {
		if unset {
			return invalidConfigMapKey(name)
		}
		if config.AuthInfos == nil {
			config.AuthInfos = make(map[string]*api.AuthInfo)
		}
		authInfo = &api.AuthInfo{}
		config.AuthInfos[name] = authInfo
	}

	switch fields[0] {
	case "client-certificate":
		return updateConfigString(&authInfo.ClientCertificate, path, nil, propertyValue, action)
	case "client-certificate-data":
		return updateConfigBytes(&authInfo.ClientCertificateData, propertyValue, action, rawBytes)
	case "client-key":
		return updateConfigString(&authInfo.ClientKey, path, nil, propertyValue, action)
	case "client-key-data":
		return updateConfigBytes(&authInfo.ClientKeyData, propertyValue, action, rawBytes)
	case "token":
		return updateConfigString(&authInfo.Token, path, nil, propertyValue, action)
	case "tokenFile":
		return updateConfigString(&authInfo.TokenFile, path, nil, propertyValue, action)
	case "act-as":
		return updateConfigString(&authInfo.Impersonate, path, nil, propertyValue, action)
	case "act-as-uid":
		return updateConfigString(&authInfo.ImpersonateUID, path, nil, propertyValue, action)
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
		return updateConfigString(&authInfo.Username, path, nil, propertyValue, action)
	case "password":
		return updateConfigString(&authInfo.Password, path, nil, propertyValue, action)
	case "auth-provider":
		if !unset {
			return unsupportedConfigProperty(path)
		}
		authInfo.AuthProvider = nil
		return nil
	case "exec":
		if !unset {
			return unsupportedConfigProperty(path)
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
		return unsupportedConfigProperty(path)
	}
}

func updateContextProperty(config *api.Config, path string, parts []string, propertyValue string, action configPropertyAction) error {
	unset := action == configPropertyUnset
	if len(parts) == 0 {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		config.Contexts = nil
		return nil
	}

	name, fields := splitConfigMapProperty(parts, contextConfigFields)
	if len(fields) == 0 {
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		if config.Contexts != nil {
			delete(config.Contexts, name)
		}
		return nil
	}
	if len(fields) != 1 {
		return unsupportedConfigProperty(path)
	}

	contextConfig := config.Contexts[name]
	if contextConfig == nil {
		if unset {
			return invalidConfigMapKey(name)
		}
		if config.Contexts == nil {
			config.Contexts = make(map[string]*api.Context)
		}
		contextConfig = &api.Context{}
		config.Contexts[name] = contextConfig
	}

	switch fields[0] {
	case "cluster":
		return updateConfigString(&contextConfig.Cluster, path, nil, propertyValue, action)
	case "user":
		return updateConfigString(&contextConfig.AuthInfo, path, nil, propertyValue, action)
	case "namespace":
		return updateConfigString(&contextConfig.Namespace, path, nil, propertyValue, action)
	case "extensions":
		if !unset {
			return fmt.Errorf("can't set a map to a value")
		}
		contextConfig.Extensions = nil
		return nil
	default:
		return unsupportedConfigProperty(path)
	}
}

func splitConfigMapProperty(parts []string, fields map[string]struct{}) (string, []string) {
	for index := 1; index < len(parts); index++ {
		if _, ok := fields[parts[index]]; ok {
			return strings.Join(parts[:index], "."), parts[index:]
		}
	}
	return strings.Join(parts, "."), nil
}

func updateConfigString(destination *string, path string, remaining []string, propertyValue string, action configPropertyAction) error {
	unset := action == configPropertyUnset
	if len(remaining) != 0 {
		return unsupportedConfigProperty(path)
	}
	if unset {
		*destination = ""
		return nil
	}
	*destination = propertyValue
	return nil
}

func updateConfigBool(destination *bool, propertyValue string, action configPropertyAction) error {
	unset := action == configPropertyUnset
	if unset {
		*destination = false
		return nil
	}
	parsed, err := parseConfigBool(propertyValue)
	if err != nil {
		return err
	}
	*destination = parsed
	return nil
}

func updateConfigBytes(destination *[]byte, propertyValue string, action configPropertyAction, rawBytes bool) error {
	unset := action == configPropertyUnset
	if unset {
		*destination = nil
		return nil
	}
	if rawBytes {
		*destination = []byte(propertyValue)
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(propertyValue)
	if err != nil {
		return fmt.Errorf("error decoding input value: %v", err)
	}
	*destination = decoded
	return nil
}

func unsupportedConfigProperty(path string) error {
	return fmt.Errorf("unable to parse one or more field values of %v", path)
}

func invalidConfigMapKey(name string) error {
	return fmt.Errorf("current map key `%v` is invalid", name)
}

func parseConfigBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	return strconv.ParseBool(value)
}
