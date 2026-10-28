package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type createRoleBindingConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	ClusterRole              string
	DryRun                   string
	FieldManager             string
	Groups                   []string
	Output                   string
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

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			clusterRoleFlag := cb.Flags.String(&cb.Config.ClusterRole)
			clusterRoleFlag = clusterRoleFlag.Description("ClusterRole this RoleBinding should reference")
			clusterRoleFlag.Long("clusterrole")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			groupValue := cb.Values.Sequence(&cb.Config.Groups, cli.Value.String())
			groupFlag := cb.Flags.Sequence(groupValue)
			groupFlag = groupFlag.Description("Groups to bind to the role. The flag can be repeated to add multiple groups.")
			groupFlag.Long("group")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			roleFlag := cb.Flags.String(&cb.Config.Role)
			roleFlag = roleFlag.Description("Role this RoleBinding should reference")
			roleFlag.Long("role")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			serviceAccountValue := cb.Values.Sequence(&cb.Config.ServiceAccounts, cli.Value.String())
			serviceAccountFlag := cb.Flags.Sequence(serviceAccountValue)
			serviceAccountFlag = serviceAccountFlag.Description("Service accounts to bind to the role, in the format <namespace>:<name>. The flag can be repeated to add multiple service accounts.")
			serviceAccountFlag.Long("serviceaccount")

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			userValue := cb.Values.Sequence(&cb.Config.Users, cli.Value.String())
			userFlag := cb.Flags.Sequence(userValue)
			userFlag = userFlag.Description("Usernames to bind to the role. The flag can be repeated to add multiple users.")
			userFlag.Long("user")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")
		},
		Run: runCreateRoleBinding,
	}
	return cmd
}
