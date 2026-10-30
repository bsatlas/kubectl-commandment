package main

import (
	"context"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createRoleBindingConfig struct {
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
	Role                     string
	SaveConfig               bool
	ServiceAccounts          []string
	ShowManagedFields        bool
	Template                 string
	Users                    []string
	Validate                 string
}

func createRoleBindingCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createRoleBindingConfig] {
	cmd := cli.Command[createRoleBindingConfig]{
		Name: "rolebinding",
		Description: cli.Description{
			Short: "Create a role binding for a particular role or cluster role",
			Long:  "Create a role binding for a particular role or cluster role.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a role binding for user1, user2, and group1 using the admin cluster role",
				CommandLine: "kubectl create rolebinding admin --clusterrole=admin --user=user1 --user=user2 --group=group1",
			},
			{
				Description: "Create a role binding for service account monitoring:sa-dev using the admin role",
				CommandLine: "kubectl create rolebinding admin-binding --role=admin --serviceaccount=monitoring:sa-dev",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createRoleBindingConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createRoleBindingConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createRoleBindingConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			cb.Rule.NotEmpty(name)

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
			clusterRoleFlag.Description("ClusterRole this RoleBinding should reference")
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
			groupFlag.Description("Groups to bind to the role. The flag can be repeated to add multiple groups.")
			groupFlag.Long("group")

			roleFlag := cb.Flags.String(&cb.Config.Role)
			cb.Values.ResolveWith(roleFlag, cli.Value.KeepLast())
			roleFlag.Description("Role this RoleBinding should reference")
			roleFlag.Long("role")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			serviceAccountValue := cb.Values.Sequence(&cb.Config.ServiceAccounts, roleBindingServiceAccountLanguage)
			serviceAccountFlag := cb.Flags.Sequence(serviceAccountValue)
			serviceAccountFlag.Description("Service accounts to bind to the role, in the format <namespace>:<name>. The flag can be repeated to add multiple service accounts.")
			serviceAccountFlag.Long("serviceaccount")

			userValue := cb.Values.Sequence(&cb.Config.Users, cli.Value.String())
			userFlag := cb.Flags.Sequence(userValue)
			userFlag.Description("Usernames to bind to the role. The flag can be repeated to add multiple users.")
			userFlag.Long("user")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Check("kubectl-create-rolebinding-role-reference", func(ctx context.Context, cfg *createRoleBindingConfig, _ cli.Dependencies) (bool, error) {
				return (cfg.ClusterRole == "") != (cfg.Role == ""), nil
			}).
				DependsOn(clusterRoleFlag.Destination().OptionalDependency()).
				DependsOn(roleFlag.Destination().OptionalDependency())
		},
		Run: runCreateRoleBinding,
	}
	return cmd
}

func runCreateRoleBinding(ctx context.Context, cfg *createRoleBindingConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewRoleBindingOptions(cfg.Streams)
	options.Name = cfg.Name
	options.ClusterRole = cfg.ClusterRole
	options.Role = cfg.Role
	options.Users = slices.Clone(cfg.Users)
	options.Groups = slices.Clone(cfg.Groups)
	options.ServiceAccounts = slices.Clone(cfg.ServiceAccounts)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	var err error
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
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
