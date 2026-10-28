package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

func configCommand(kubeconfig, contextName cli.Destination[rootConfig, string], client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "config",
		Description: cli.Description{
			Short: "Modify kubeconfig files",
			Long:  "Modify kubeconfig files using subcommands like \"kubectl config set current-context my-context\".\n\n The loading order follows these rules:\n\n  1.  If the --kubeconfig flag is set, then only that file is loaded. The flag may only be set once and no merging takes place.\n  2.  If $KUBECONFIG environment variable is set, then it is used as a list of paths (normal path delimiting rules for your system). These paths are merged. When a value is modified, it is modified in the file that defines the stanza. When a value is created, it is created in the first file that exists. If no files in the chain exist, then it creates the last file in the list.\n  3.  Otherwise, ${HOME}/.kube/config is used and no merging takes place.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			currentContext := currentContextCommand(kubeconfig)
			cb.Subcommands.Command(currentContext)

			deleteCluster := configDeleteClusterCommand(kubeconfig)
			cb.Subcommands.Command(deleteCluster)

			deleteContext := configDeleteContextCommand(kubeconfig)
			cb.Subcommands.Command(deleteContext)

			deleteUser := configDeleteUserCommand(kubeconfig)
			cb.Subcommands.Command(deleteUser)

			getClusters := configGetClustersCommand(kubeconfig)
			cb.Subcommands.Command(getClusters)

			getContexts := configGetContextsCommand(kubeconfig)
			cb.Subcommands.Command(getContexts)

			getUsers := configGetUsersCommand(kubeconfig)
			cb.Subcommands.Command(getUsers)

			renameContext := configRenameContextCommand(kubeconfig)
			cb.Subcommands.Command(renameContext)

			set := configSetCommand(kubeconfig)
			cb.Subcommands.Command(set)

			setCluster := configSetClusterCommand(kubeconfig)
			cb.Subcommands.Command(setCluster)

			setContext := configSetContextCommand(kubeconfig, client)
			cb.Subcommands.Command(setContext)

			setCredentials := configSetCredentialsCommand(kubeconfig)
			cb.Subcommands.Command(setCredentials)

			unset := configUnsetCommand(kubeconfig)
			cb.Subcommands.Command(unset)

			useContext := useContextCommand(kubeconfig)
			cb.Subcommands.Command(useContext)

			view := configViewCommand(kubeconfig, contextName)
			cb.Subcommands.Command(view)
		},
	}
	return cmd
}

func deriveKubeconfig[C any](command *cli.CommandBuilder[C], storage *string, source cli.Destination[rootConfig, string]) cli.Destination[C, string] {
	return command.Values.Derive(storage, cli.Value.ComputedBy(func(_ context.Context, _ *C, dependencies cli.Dependencies) (string, error) {
		value, ok := dependencies.Get(source)
		if !ok {
			return "", errors.New("kubeconfig dependency is unavailable")
		}
		return value, nil
	}).From(source))
}

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

type configProperty struct {
	Path     string
	Root     configPropertyRoot
	Name     string
	Field    string
	HasName  bool
	HasField bool
}

func isClusterConfigField(name string) bool {
	switch name {
	case "server",
		"tls-server-name",
		"insecure-skip-tls-verify",
		"certificate-authority",
		"certificate-authority-data",
		"proxy-url",
		"disable-compression",
		"extensions":
		return true
	default:
		return false
	}
}

func isAuthInfoConfigField(name string) bool {
	switch name {
	case "client-certificate",
		"client-certificate-data",
		"client-key",
		"client-key-data",
		"token",
		"tokenFile",
		"act-as",
		"act-as-uid",
		"act-as-groups",
		"act-as-user-extra",
		"username",
		"password",
		"auth-provider",
		"exec",
		"extensions":
		return true
	default:
		return false
	}
}

func isContextConfigField(name string) bool {
	switch name {
	case "cluster", "user", "namespace", "extensions":
		return true
	default:
		return false
	}
}

func isPreferencesConfigField(name string) bool {
	switch name {
	case "colors", "extensions":
		return true
	default:
		return false
	}
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

	property := configProperty{
		Path: path,
	}
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
		if err := parseConfigStructProperty(&property, parts[1:], isPreferencesConfigField); err != nil {
			return configProperty{}, err
		}
	case "clusters":
		property.Root = configPropertyRootClusters
		if err := parseConfigMapProperty(&property, parts[1:], isClusterConfigField); err != nil {
			return configProperty{}, err
		}
	case "users":
		property.Root = configPropertyRootUsers
		if err := parseConfigMapProperty(&property, parts[1:], isAuthInfoConfigField); err != nil {
			return configProperty{}, err
		}
	case "contexts":
		property.Root = configPropertyRootContexts
		if err := parseConfigMapProperty(&property, parts[1:], isContextConfigField); err != nil {
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

func parseConfigStructProperty(property *configProperty, parts []string, isField func(string) bool) error {
	if len(parts) == 0 {
		return nil
	}
	if len(parts) != 1 {
		return unsupportedConfigProperty(property.Path)
	}
	if !isField(parts[0]) {
		return unsupportedConfigProperty(property.Path)
	}
	property.Field = parts[0]
	property.HasField = true
	return nil
}

func parseConfigMapProperty(property *configProperty, parts []string, isField func(string) bool) error {
	if len(parts) == 0 {
		return nil
	}
	property.HasName = true
	for index := 1; index < len(parts); index++ {
		if !isField(parts[index]) {
			continue
		}
		if index+1 != len(parts) {
			return unsupportedConfigProperty(property.Path)
		}
		property.Name = strings.Join(parts[:index], ".")
		property.Field = parts[index]
		property.HasField = true
		return nil
	}
	property.Name = strings.Join(parts, ".")
	return nil
}
