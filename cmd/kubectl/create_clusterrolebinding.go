package main

import (
	"context"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createClusterRoleBindingConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	ClusterRole              string
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	Groups                   []string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	SaveConfig               bool
	ServiceAccounts          []string
	ShowManagedFields        bool
	Template                 string
	Users                    []string
	Validate                 string
}

func createClusterRoleBindingCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createClusterRoleBindingConfig] {
	cmd := cli.Command[createClusterRoleBindingConfig]{
		Name: "clusterrolebinding",
		Description: cli.Description{
			Short: "Create a cluster role binding for a particular cluster role",
			Long:  "Create a cluster role binding for a particular cluster role.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a cluster role binding for user1, user2, and group1 using the cluster-admin cluster role",
				CommandLine: "kubectl create clusterrolebinding cluster-admin --clusterrole=cluster-admin --user=user1 --user=user2 --group=group1",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createClusterRoleBindingConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createClusterRoleBindingConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createClusterRoleBindingConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			clusterRoleFlag := cb.Flags.String(&cb.Config.ClusterRole)
			cb.Values.ResolveWith(clusterRoleFlag, cli.Value.KeepLast())
			clusterRoleFlag.Description("ClusterRole this ClusterRoleBinding should reference")
			clusterRoleFlag.CompleteBy(fixedResourceArgumentCompleter(
				func(cfg *createClusterRoleBindingConfig) (clientConfig, []string) { return cfg.Client, nil },
				"clusterrole",
			)).From(client)
			clusterRoleFlag.Required()
			cb.Rule.Require(clusterRoleFlag.Destination())
			clusterRoleFlag.Long("clusterrole")

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.").Long("dry-run").ImplicitValue(cmdutil.DryRunClient)
			cb.Values.Default(dryRun, cmdutil.DryRunNone)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			groupValue := cb.Values.Sequence(&cb.Config.Groups, cli.Value.String())
			groupFlag := cb.Flags.Sequence(groupValue)
			groupFlag.Description("Groups to bind to the clusterrole. The flag can be repeated to add multiple groups.")
			groupFlag.Long("group")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			serviceAccountValue := cb.Values.Sequence(&cb.Config.ServiceAccounts, roleBindingServiceAccountLanguage)
			serviceAccountFlag := cb.Flags.Sequence(serviceAccountValue)
			serviceAccountFlag.Description("Service accounts to bind to the clusterrole, in the format <namespace>:<name>. The flag can be repeated to add multiple service accounts.")
			serviceAccountFlag.Long("serviceaccount")

			userValue := cb.Values.Sequence(&cb.Config.Users, cli.Value.String())
			userFlag := cb.Flags.Sequence(userValue)
			userFlag.Description("Usernames to bind to the clusterrole. The flag can be repeated to add multiple users.")
			userFlag.Long("user")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")
		},
		Run: runCreateClusterRoleBinding,
	}
	return cmd
}

func runCreateClusterRoleBinding(ctx context.Context, cfg *createClusterRoleBindingConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewClusterRoleBindingOptions(cfg.Streams)
	options.Name = cfg.Name
	options.ClusterRole = cfg.ClusterRole
	options.Users = slices.Clone(cfg.Users)
	options.Groups = slices.Clone(cfg.Groups)
	options.ServiceAccounts = slices.Clone(cfg.ServiceAccounts)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.RbacV1()
	options.DryRunStrategy = cfg.DryRun
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlagsWithOutputState(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}
