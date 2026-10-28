package main

import "github.com/abstractionsinc/commandment/cli"

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
