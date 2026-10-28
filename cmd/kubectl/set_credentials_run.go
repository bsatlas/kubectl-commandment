package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func runConfigSetCredentials(_ context.Context, cfg *configSetCredentialsConfig) error {
	if cfg.EmbedCerts {
		if cfg.ClientCertificate != "" {
			if _, err := os.Stat(cfg.ClientCertificate); err != nil {
				return fmt.Errorf("could not stat %s file %s: %v", clientcmd.FlagCertFile, cfg.ClientCertificate, err)
			}
		}
		if cfg.ClientKey != "" {
			if _, err := os.Stat(cfg.ClientKey); err != nil {
				return fmt.Errorf("could not stat %s file %s: %v", clientcmd.FlagKeyFile, cfg.ClientKey, err)
			}
		}
	}
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	starting, exists := config.AuthInfos[cfg.Name]
	if !exists {
		starting = api.NewAuthInfo()
	}
	modified, err := modifyConfigAuthInfo(*starting, cfg)
	if err != nil {
		return err
	}
	config.AuthInfos[cfg.Name] = &modified
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "User %q set.\n", cfg.Name)
	return err
}

func modifyConfigAuthInfo(existing api.AuthInfo, cfg *configSetCredentialsConfig) (api.AuthInfo, error) {
	modified := existing

	if flagProvided(cfg.ClientCertificateState) {
		path := cfg.ClientCertificate
		if cfg.EmbedCerts {
			data, err := os.ReadFile(path)
			if err != nil {
				return api.AuthInfo{}, err
			}
			modified.ClientCertificateData = data
			modified.ClientCertificate = ""
		} else {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return api.AuthInfo{}, err
			}
			modified.ClientCertificate = absolute
			if absolute != "" {
				modified.ClientCertificateData = nil
			}
		}
	}
	if flagProvided(cfg.ClientKeyState) {
		path := cfg.ClientKey
		if cfg.EmbedCerts {
			data, err := os.ReadFile(path)
			if err != nil {
				return api.AuthInfo{}, err
			}
			modified.ClientKeyData = data
			modified.ClientKey = ""
		} else {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return api.AuthInfo{}, err
			}
			modified.ClientKey = absolute
			if absolute != "" {
				modified.ClientKeyData = nil
			}
		}
	}

	setToken := false
	setBasic := false
	if flagProvided(cfg.TokenState) {
		modified.Token = cfg.Token
		setToken = modified.Token != ""
	}
	if flagProvided(cfg.UsernameState) {
		modified.Username = cfg.Username
		setBasic = modified.Username != ""
	}
	if flagProvided(cfg.PasswordState) {
		modified.Password = cfg.Password
		setBasic = setBasic || modified.Password != ""
	}

	if flagProvided(cfg.AuthProviderState) {
		if modified.AuthProvider == nil || modified.AuthProvider.Name != cfg.AuthProvider {
			modified.AuthProvider = &api.AuthProviderConfig{
				Name: cfg.AuthProvider,
			}
		}
	}
	if modified.AuthProvider != nil {
		if modified.AuthProvider.Config == nil {
			modified.AuthProvider.Config = make(map[string]string)
		}
		pairs := cfg.AuthProviderArg
		for _, pair := range pairs {
			if pair.Remove {
				delete(modified.AuthProvider.Config, pair.Name)
			}
		}
		for _, pair := range pairs {
			if !pair.Remove {
				modified.AuthProvider.Config[pair.Name] = pair.Value
			}
		}
	}

	if flagProvided(cfg.ExecCommandState) {
		if modified.Exec == nil {
			modified.Exec = &api.ExecConfig{
				Command: cfg.ExecCommand,
			}
		} else {
			modified.Exec.Command = cfg.ExecCommand
			modified.Exec.Args = nil
		}
	}
	if modified.Exec != nil {
		if flagProvided(cfg.ExecAPIVersionState) {
			modified.Exec.APIVersion = cfg.ExecAPIVersion
		}
		if len(cfg.ExecArg) > 0 {
			modified.Exec.Args = slices.Clone(cfg.ExecArg)
		}
		if flagProvided(cfg.ExecInteractiveState) {
			modified.Exec.InteractiveMode = cfg.ExecInteractiveMode
		}
		if flagProvided(cfg.ExecProvideClusterState) {
			modified.Exec.ProvideClusterInfo = cfg.ExecProvideClusterInfo
		}
		if len(cfg.ExecEnv) > 0 {
			remove := make(map[string]struct{})
			for _, pair := range cfg.ExecEnv {
				if pair.Remove {
					remove[pair.Name] = struct{}{}
				}
			}
			kept := modified.Exec.Env[:0]
			for _, variable := range modified.Exec.Env {
				if _, found := remove[variable.Name]; !found {
					kept = append(kept, variable)
				}
			}
			modified.Exec.Env = kept
		}
		for _, pair := range cfg.ExecEnv {
			if pair.Remove {
				continue
			}
			updated := false
			for index := range modified.Exec.Env {
				if modified.Exec.Env[index].Name == pair.Name {
					modified.Exec.Env[index].Value = pair.Value
					updated = true
					break
				}
			}
			if !updated {
				variable := api.ExecEnvVar{
					Name:  pair.Name,
					Value: pair.Value,
				}
				modified.Exec.Env = append(modified.Exec.Env, variable)
			}
		}
	}

	if setToken || setBasic {
		if !setToken {
			modified.Token = ""
		}
		if !setBasic {
			modified.Username = ""
			modified.Password = ""
		}
	}
	return modified, nil
}
