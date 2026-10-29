package main

import (
	"context"
	"fmt"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type rootConfig struct {
	CacheDir             string
	Kubeconfig           string
	Cluster              string
	User                 string
	Context              string
	Namespace            string
	Server               string
	TLSServerName        string
	Insecure             bool
	ClientCertificate    string
	ClientKey            string
	CertificateAuthority string
	Token                string
	Impersonate          string
	ImpersonateUID       string
	ImpersonateGroups    []string
	ImpersonateUserExtra []string
	Username             string
	Password             string
	RequestTimeout       string
	DisableCompression   bool
	ProxyURL             string

	Kuberc             string
	MatchServerVersion bool
	Profile            string
	ProfileOutput      string
	WarningsAsErrors   bool

	Client clientConfig
}

func application(plugins pluginCatalog) (cli.Application, error) {
	return cli.Compile(applicationDefinition(plugins))
}

func applicationDefinition(plugins pluginCatalog) cli.Definition[rootConfig] {
	definition := cli.Definition[rootConfig]{
		Name:    "kubectl",
		Release: kubectlRelease,
		Description: cli.Description{
			Short: "kubectl controls the Kubernetes cluster manager",
			Long:  "kubectl controls the Kubernetes cluster manager.\n\n Find more information at: https://kubernetes.io/docs/reference/kubectl/",
		},
		Help: cli.HelpConfig{
			Renderer: renderHelp,
		},
		ShellCompletion: cli.ShellCompletion{
			Shells: []cli.ShellCompletionProvider{
				cli.Shell.Bash,
				cli.Shell.Zsh,
				cli.Shell.Fish,
				cli.Shell.PowerShell,
			},
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			root.Dialect.ShortFlagClustering(cli.ShortFlagClusteringEnabled)

			defaults := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()

			cacheDir := root.Values.String(&root.Config.CacheDir)
			root.Values.Default(cacheDir, *defaults.CacheDir)
			root.Flags.ValueFor(cacheDir).Placement(cli.FlagPlacementSubtree).Description("Default cache directory").Long("cache-dir")

			kubeconfig := root.Values.String(&root.Config.Kubeconfig)
			root.Values.Default(kubeconfig, *defaults.KubeConfig)
			root.Flags.ValueFor(kubeconfig).Placement(cli.FlagPlacementSubtree).Description("Path to the kubeconfig file to use for CLI requests.").Long("kubeconfig")

			cluster := root.Values.String(&root.Config.Cluster)
			root.Values.Default(cluster, *defaults.ClusterName)
			clusterFlag := root.Flags.ValueFor(cluster).Placement(cli.FlagPlacementSubtree).Description("The name of the kubeconfig cluster to use")
			clusterFlag.CompleteBy(kubeconfigNameCompleter(
				func(cfg *rootConfig) string { return cfg.Kubeconfig },
				kubeconfigClusterName,
			)).From(kubeconfig)
			clusterFlag.Long("cluster")

			user := root.Values.String(&root.Config.User)
			root.Values.Default(user, *defaults.AuthInfoName)
			userFlag := root.Flags.ValueFor(user).Placement(cli.FlagPlacementSubtree).Description("The name of the kubeconfig user to use")
			userFlag.CompleteBy(kubeconfigNameCompleter(
				func(cfg *rootConfig) string { return cfg.Kubeconfig },
				kubeconfigUserName,
			)).From(kubeconfig)
			userFlag.Long("user")

			contextName := root.Values.String(&root.Config.Context)
			root.Values.Default(contextName, *defaults.Context)
			contextFlag := root.Flags.ValueFor(contextName).Placement(cli.FlagPlacementSubtree).Description("The name of the kubeconfig context to use")
			contextFlag.CompleteBy(kubeconfigNameCompleter(
				func(cfg *rootConfig) string { return cfg.Kubeconfig },
				kubeconfigContextName,
			)).From(kubeconfig)
			contextFlag.Long("context")

			namespace := root.Values.String(&root.Config.Namespace)
			root.Values.Default(namespace, *defaults.Namespace)
			namespaceFlag := root.Flags.ValueFor(namespace).Placement(cli.FlagPlacementSubtree).Description("If present, the namespace scope for this CLI request")
			namespaceFlag.Long("namespace")
			namespaceFlag.Short('n')

			server := root.Values.String(&root.Config.Server)
			root.Values.Default(server, *defaults.APIServer)
			serverFlag := root.Flags.ValueFor(server).Placement(cli.FlagPlacementSubtree).Description("The address and port of the Kubernetes API server")
			serverFlag.Long("server")
			serverFlag.Short('s')

			tlsServerName := root.Values.String(&root.Config.TLSServerName)
			root.Values.Default(tlsServerName, *defaults.TLSServerName)
			root.Flags.ValueFor(tlsServerName).Placement(cli.FlagPlacementSubtree).Description("Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used").Long("tls-server-name")

			insecure := root.Values.Bool(&root.Config.Insecure)
			root.Values.Default(insecure, *defaults.Insecure)
			root.Flags.ValueFor(insecure).Placement(cli.FlagPlacementSubtree).Description("If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure").Long("insecure-skip-tls-verify").ImplicitValue(true)

			clientCertificate := root.Values.String(&root.Config.ClientCertificate)
			root.Values.Default(clientCertificate, *defaults.CertFile)
			root.Flags.ValueFor(clientCertificate).Placement(cli.FlagPlacementSubtree).Description("Path to a client certificate file for TLS").Long("client-certificate")

			clientKey := root.Values.String(&root.Config.ClientKey)
			root.Values.Default(clientKey, *defaults.KeyFile)
			root.Flags.ValueFor(clientKey).Placement(cli.FlagPlacementSubtree).Description("Path to a client key file for TLS").Long("client-key")

			certificateAuthority := root.Values.String(&root.Config.CertificateAuthority)
			root.Values.Default(certificateAuthority, *defaults.CAFile)
			root.Flags.ValueFor(certificateAuthority).Placement(cli.FlagPlacementSubtree).Description("Path to a cert file for the certificate authority").Long("certificate-authority")

			token := root.Values.String(&root.Config.Token)
			root.Values.Default(token, *defaults.BearerToken)
			root.Flags.ValueFor(token).Placement(cli.FlagPlacementSubtree).Description("Bearer token for authentication to the API server").Long("token").Sensitive()

			impersonate := root.Values.String(&root.Config.Impersonate)
			root.Values.Default(impersonate, *defaults.Impersonate)
			root.Flags.ValueFor(impersonate).Placement(cli.FlagPlacementSubtree).Description("Username to impersonate for the operation. User could be a regular user or a service account in a namespace.").Long("as")

			impersonateUID := root.Values.String(&root.Config.ImpersonateUID)
			root.Values.Default(impersonateUID, *defaults.ImpersonateUID)
			root.Flags.ValueFor(impersonateUID).Placement(cli.FlagPlacementSubtree).Description("UID to impersonate for the operation.").Long("as-uid")

			impersonateGroups := root.Values.Sequence(&root.Config.ImpersonateGroups, cli.Value.String())
			groupFlag := root.Flags.Sequence(impersonateGroups).Placement(cli.FlagPlacementSubtree).Description("Group to impersonate for the operation, this flag can be repeated to specify multiple groups.")
			groupFlag.Occurrences(cli.Count.ZeroOrMore()).ValuesPerOccurrence(cli.Count.Exactly(1)).Long("as-group")

			impersonateUserExtra := root.Values.Sequence(&root.Config.ImpersonateUserExtra, cli.Value.String())
			extraFlag := root.Flags.Sequence(impersonateUserExtra).Placement(cli.FlagPlacementSubtree).Description("User extras to impersonate for the operation, this flag can be repeated to specify multiple values for the same key.")
			extraFlag.Occurrences(cli.Count.ZeroOrMore()).ValuesPerOccurrence(cli.Count.Exactly(1)).Long("as-user-extra")

			username := root.Values.String(&root.Config.Username)
			root.Values.Default(username, "")
			root.Flags.ValueFor(username).Placement(cli.FlagPlacementSubtree).Description("Username for basic authentication to the API server").Long("username")

			password := root.Values.String(&root.Config.Password)
			root.Values.Default(password, "")
			root.Flags.ValueFor(password).Placement(cli.FlagPlacementSubtree).Description("Password for basic authentication to the API server").Long("password").Sensitive()

			requestTimeout := root.Values.String(&root.Config.RequestTimeout)
			root.Values.Default(requestTimeout, *defaults.Timeout)
			root.Flags.ValueFor(requestTimeout).Placement(cli.FlagPlacementSubtree).Description("The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests.").Long("request-timeout")

			disableCompression := root.Values.Bool(&root.Config.DisableCompression)
			root.Values.Default(disableCompression, *defaults.DisableCompression)
			root.Flags.ValueFor(disableCompression).Placement(cli.FlagPlacementSubtree).Description("If true, opt-out of response compression for all requests to the server").Long("disable-compression").ImplicitValue(true)

			proxyURL := root.Values.String(&root.Config.ProxyURL)
			root.Values.Default(proxyURL, *defaults.ProxyURL)
			root.Flags.ValueFor(proxyURL).Placement(cli.FlagPlacementSubtree).Description("Proxy URL to use for requests to the API server").Long("proxy-url")

			namespaceFlag.CompleteBy(completeRootNamespace).From(
				cacheDir, kubeconfig, cluster, user, contextName, server, tlsServerName, insecure,
				clientCertificate, clientKey, certificateAuthority, token, impersonate, impersonateUID,
				impersonateGroups, impersonateUserExtra, username, password, requestTimeout, disableCompression, proxyURL,
			)

			kuberc := root.Values.String(&root.Config.Kuberc)
			root.Values.Default(kuberc, "")
			root.Flags.ValueFor(kuberc).Placement(cli.FlagPlacementSubtree).Description("Path to the kuberc file to use for preferences. This can be disabled by exporting KUBECTL_KUBERC=false feature gate or turning off the feature KUBERC=off.").Long("kuberc")

			matchServerVersion := root.Values.Bool(&root.Config.MatchServerVersion)
			root.Values.Default(matchServerVersion, false)
			root.Flags.ValueFor(matchServerVersion).Placement(cli.FlagPlacementSubtree).Description("Require server version to match client version").Long("match-server-version").ImplicitValue(true)

			profile := root.Values.String(&root.Config.Profile)
			root.Values.Default(profile, "none")
			root.Flags.ValueFor(profile).Placement(cli.FlagPlacementSubtree).Description("Name of profile to capture. One of (none|cpu|heap|goroutine|threadcreate|block|mutex|trace)").Long("profile")

			profileOutput := root.Values.String(&root.Config.ProfileOutput)
			root.Values.Default(profileOutput, "profile.pprof")
			root.Flags.ValueFor(profileOutput).Placement(cli.FlagPlacementSubtree).Description("Name of the file to write the profile to").Long("profile-output")

			warningsAsErrors := root.Values.Bool(&root.Config.WarningsAsErrors)
			root.Values.Default(warningsAsErrors, false)
			root.Flags.ValueFor(warningsAsErrors).Placement(cli.FlagPlacementSubtree).Description("Treat warnings received from the server as errors and exit with a non-zero exit code").Long("warnings-as-errors").ImplicitValue(true)

			client := root.Values.Derive(&root.Config.Client, cli.Value.ComputedBy(func(_ context.Context, _ *rootConfig, dependencies cli.Dependencies) (clientConfig, error) {
				cacheDirValue, ok := dependencies.Get(cacheDir)
				if !ok {
					return clientConfig{}, fmt.Errorf("cache directory dependency is unavailable")
				}
				kubeconfigValue, ok := dependencies.Get(kubeconfig)
				if !ok {
					return clientConfig{}, fmt.Errorf("kubeconfig dependency is unavailable")
				}
				clusterValue, ok := dependencies.Get(cluster)
				if !ok {
					return clientConfig{}, fmt.Errorf("cluster dependency is unavailable")
				}
				userValue, ok := dependencies.Get(user)
				if !ok {
					return clientConfig{}, fmt.Errorf("user dependency is unavailable")
				}
				contextValue, ok := dependencies.Get(contextName)
				if !ok {
					return clientConfig{}, fmt.Errorf("context dependency is unavailable")
				}
				namespaceValue, ok := dependencies.Get(namespace)
				if !ok {
					return clientConfig{}, fmt.Errorf("namespace dependency is unavailable")
				}
				serverValue, ok := dependencies.Get(server)
				if !ok {
					return clientConfig{}, fmt.Errorf("server dependency is unavailable")
				}
				tlsServerNameValue, ok := dependencies.Get(tlsServerName)
				if !ok {
					return clientConfig{}, fmt.Errorf("TLS server name dependency is unavailable")
				}
				insecureValue, ok := dependencies.Get(insecure)
				if !ok {
					return clientConfig{}, fmt.Errorf("TLS verification dependency is unavailable")
				}
				clientCertificateValue, ok := dependencies.Get(clientCertificate)
				if !ok {
					return clientConfig{}, fmt.Errorf("client certificate dependency is unavailable")
				}
				clientKeyValue, ok := dependencies.Get(clientKey)
				if !ok {
					return clientConfig{}, fmt.Errorf("client key dependency is unavailable")
				}
				certificateAuthorityValue, ok := dependencies.Get(certificateAuthority)
				if !ok {
					return clientConfig{}, fmt.Errorf("certificate authority dependency is unavailable")
				}
				tokenValue, ok := dependencies.Get(token)
				if !ok {
					return clientConfig{}, fmt.Errorf("token dependency is unavailable")
				}
				impersonateValue, ok := dependencies.Get(impersonate)
				if !ok {
					return clientConfig{}, fmt.Errorf("impersonation dependency is unavailable")
				}
				impersonateUIDValue, ok := dependencies.Get(impersonateUID)
				if !ok {
					return clientConfig{}, fmt.Errorf("impersonation UID dependency is unavailable")
				}
				impersonateGroupsValue, ok := dependencies.Get(impersonateGroups)
				if !ok {
					return clientConfig{}, fmt.Errorf("impersonation groups dependency is unavailable")
				}
				impersonateUserExtraValue, ok := dependencies.Get(impersonateUserExtra)
				if !ok {
					return clientConfig{}, fmt.Errorf("impersonation extras dependency is unavailable")
				}
				usernameValue, ok := dependencies.Get(username)
				if !ok {
					return clientConfig{}, fmt.Errorf("username dependency is unavailable")
				}
				passwordValue, ok := dependencies.Get(password)
				if !ok {
					return clientConfig{}, fmt.Errorf("password dependency is unavailable")
				}
				requestTimeoutValue, ok := dependencies.Get(requestTimeout)
				if !ok {
					return clientConfig{}, fmt.Errorf("request timeout dependency is unavailable")
				}
				disableCompressionValue, ok := dependencies.Get(disableCompression)
				if !ok {
					return clientConfig{}, fmt.Errorf("compression dependency is unavailable")
				}
				proxyURLValue, ok := dependencies.Get(proxyURL)
				if !ok {
					return clientConfig{}, fmt.Errorf("proxy URL dependency is unavailable")
				}

				client := clientConfig{
					CacheDir:             cacheDirValue,
					Kubeconfig:           kubeconfigValue,
					Cluster:              clusterValue,
					User:                 userValue,
					Context:              contextValue,
					Namespace:            namespaceValue,
					Server:               serverValue,
					TLSServerName:        tlsServerNameValue,
					Insecure:             insecureValue,
					ClientCertificate:    clientCertificateValue,
					ClientKey:            clientKeyValue,
					CertificateAuthority: certificateAuthorityValue,
					Token:                tokenValue,
					Impersonate:          impersonateValue,
					ImpersonateUID:       impersonateUIDValue,
					ImpersonateGroups:    impersonateGroupsValue,
					ImpersonateUserExtra: impersonateUserExtraValue,
					Username:             usernameValue,
					Password:             passwordValue,
					RequestTimeout:       requestTimeoutValue,
					DisableCompression:   disableCompressionValue,
					ProxyURL:             proxyURLValue,
				}
				return client, nil
			}).From(cacheDir, kubeconfig, cluster, user, contextName, namespace, server, tlsServerName, insecure, clientCertificate, clientKey, certificateAuthority, token, impersonate, impersonateUID, impersonateGroups, impersonateUserExtra, username, password, requestTimeout, disableCompression, proxyURL))

			root.Subcommands.Command(alphaCommand)
			annotate := annotateCommand(client)
			root.Subcommands.Command(annotate)
			apiResources := apiResourcesCommand(client)
			root.Subcommands.Command(apiResources)
			apiVersions := apiVersionsCommand(client)
			root.Subcommands.Command(apiVersions)
			apply := applyCommand(client)
			root.Subcommands.Command(apply)
			attach := attachCommand(client)
			root.Subcommands.Command(attach)
			auth := authCommand(client)
			root.Subcommands.Command(auth)
			autoscale := autoscaleCommand(client)
			root.Subcommands.Command(autoscale)
			certificate := certificateCommand(client)
			root.Subcommands.Command(certificate)
			clusterInfo := clusterInfoCommand(client)
			root.Subcommands.Command(clusterInfo)

			config := configCommand(kubeconfig, contextName, client)
			root.Subcommands.Command(config)

			cordon := cordonCommand(client)
			root.Subcommands.Command(cordon)
			cp := cpCommand(client)
			root.Subcommands.Command(cp)
			create := createCommand(client, plugins.create)
			root.Subcommands.Command(create)
			debug := debugCommand(client)
			root.Subcommands.Command(debug)
			deleteDefinition := deleteCommand(client)
			root.Subcommands.Command(deleteDefinition)
			describe := describeCommand(client)
			root.Subcommands.Command(describe)
			diff := diffCommand(client)
			root.Subcommands.Command(diff)
			drain := drainCommand(client)
			root.Subcommands.Command(drain)
			edit := editCommand(client)
			root.Subcommands.Command(edit)
			events := eventsCommand(client)
			root.Subcommands.Command(events)
			exec := execCommand(client)
			root.Subcommands.Command(exec)
			explain := explainCommand(client)
			root.Subcommands.Command(explain)
			expose := exposeCommand(client)
			root.Subcommands.Command(expose)

			get := getCommand(client)
			root.Subcommands.Command(get)

			root.Subcommands.Command(kubercCommand)
			root.Subcommands.Command(kustomizeCommand)
			label := labelCommand(client)
			root.Subcommands.Command(label)
			logs := logsCommand(client)
			root.Subcommands.Command(logs)
			root.Subcommands.Command(optionsCommand)
			patch := patchCommand(client)
			root.Subcommands.Command(patch)
			root.Subcommands.Command(pluginCommand)
			portForward := portForwardCommand(client)
			root.Subcommands.Command(portForward)
			proxy := proxyCommand(client)
			root.Subcommands.Command(proxy)
			replace := replaceCommand(client)
			root.Subcommands.Command(replace)
			rollout := rolloutCommand(client)
			root.Subcommands.Command(rollout)
			run := runCommand(client)
			root.Subcommands.Command(run)
			scale := scaleCommand(client)
			root.Subcommands.Command(scale)
			set := setCommand(client)
			root.Subcommands.Command(set)
			taint := taintCommand(client)
			root.Subcommands.Command(taint)
			top := topCommand(client)
			root.Subcommands.Command(top)
			uncordon := uncordonCommand(client)
			root.Subcommands.Command(uncordon)
			version := versionCommand(client, warningsAsErrors)
			root.Subcommands.Command(version)
			wait := waitCommand(client)
			root.Subcommands.Command(wait)

			occupied := pluginOccupiedNames(
				alphaCommand.Name, annotate.Name, apiResources.Name, apiVersions.Name, apply.Name, attach.Name, auth.Name,
				autoscale.Name, certificate.Name, clusterInfo.Name, config.Name, cordon.Name, cp.Name, create.Name, debug.Name,
				delete.Name, describe.Name, diff.Name, drain.Name, edit.Name, events.Name, exec.Name, explain.Name, expose.Name,
				get.Name, kubercCommand.Name, kustomizeCommand.Name, label.Name, logs.Name, optionsCommand.Name, patch.Name,
				pluginCommand.Name, portForward.Name, proxy.Name, replace.Name, rollout.Name, run.Name, scale.Name, set.Name,
				taint.Name, top.Name, uncordon.Name, version.Name, wait.Name,
			)
			addPluginProviders(&root.Subcommands, plugins.root, occupied)
		},
	}
	return definition
}
