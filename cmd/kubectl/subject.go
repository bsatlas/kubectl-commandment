package main

import (
	"context"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type setSubjectConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                []string
	All                      bool
	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	FieldManager             string
	Filename                 []string
	Group                    []string
	Kustomize                string
	Local                    bool
	Output                   string
	Recursive                bool
	Selector                 string
	ServiceAccounts          []subjectServiceAccount
	ShowManagedFields        bool
	Template                 string
	User                     []string
}

type subjectServiceAccount struct {
	namespace string
	name      string
}

var subjectServiceAccountLanguage = cli.ValueCodec[subjectServiceAccount]{
	Name: "kubectl-subject-service-account",
	Parse: func(value string) (subjectServiceAccount, error) {
		namespace, name, found := strings.Cut(value, ":")
		if !found || name == "" || strings.Contains(name, ":") {
			return subjectServiceAccount{}, cli.Rejection.Message(
				"serviceaccount",
				"serviceaccount must be <namespace>:<name>",
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		serviceAccount := subjectServiceAccount{
			namespace: namespace,
			name:      name,
		}
		return serviceAccount, nil
	},
	Format: func(value subjectServiceAccount) string {
		return value.namespace + ":" + value.name
	},
	Snapshot: cli.Value.Assign[subjectServiceAccount](),
}

func setSubjectCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setSubjectConfig] {
	cmd := cli.Command[setSubjectConfig]{
		Name: "subject",
		Description: cli.Description{
			Short: "Update the user, group, or service account in a role binding or cluster role binding",
			Long:  "Update the user, group, or service account in a role binding or cluster role binding.",
		},
		Examples: []cli.Example{
			{
				Description: "Update a cluster role binding for serviceaccount1",
				CommandLine: "kubectl set subject clusterrolebinding admin --serviceaccount=namespace:serviceaccount1",
			},
			{
				Description: "Update a role binding for user1, user2, and group1",
				CommandLine: "kubectl set subject rolebinding admin --user=user1 --user=user2 --group=group1",
			},
			{
				Description: "Print the result (in YAML format) of updating rolebinding subjects from a local, without hitting the server",
				CommandLine: "kubectl create rolebinding admin --role=admin --user=admin -o yaml --dry-run=client | kubectl set subject --local -f - --user=foo -o yaml",
			},
		},
		Declare: func(cb *cli.CommandBuilder[setSubjectConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			resources := cb.Values.Sequence(&cb.Config.Resources, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setSubjectConfig]) {
				form.Sequence(resources, "RESOURCE", cli.Count.ZeroOrMore())
			})

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			allFlag.Description("Select all resources, in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, util.DryRunNone)
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue(util.DryRunClient)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-set")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("Filename, directory, or URL to files the resource to update the subjects")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			groupValue := cb.Values.Sequence(&cb.Config.Group, cli.Value.String())
			groupFlag := cb.Flags.Sequence(groupValue)
			groupFlag.Description("Groups to bind to the role")
			groupFlag.Long("group")

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			localFlag.Description("If true, set subject will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recursive := cb.Values.Bool(&cb.Config.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			serviceAccountValue := cb.Values.Sequence(&cb.Config.ServiceAccounts, subjectServiceAccountLanguage)
			serviceAccountFlag := cb.Flags.Sequence(serviceAccountValue)
			serviceAccountFlag.Description("Service accounts to bind to the role")
			serviceAccountFlag.Long("serviceaccount")

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			userValue := cb.Values.Sequence(&cb.Config.User, cli.Value.String())
			userFlag := cb.Flags.Sequence(userValue)
			userFlag.Description("Usernames to bind to the role")
			userFlag.Long("user")

			cb.Rule.When(cb.When.Equal(all, true)).Equal(selector, "")
			cb.Rule.AtMostOne(filenameValue, kustomize)
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.When(cb.When.Equal(local, true)).Forbid(resources)
			cb.Rule.AtLeastOne(resources, filenameValue).And(kustomize)
			cb.Rule.AtLeastOne(userValue, groupValue).And(serviceAccountValue)
			cb.Rule.Check("kubectl-set-subject-local-dry-run", func(_ context.Context, cfg *setSubjectConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.Local || cfg.DryRun != util.DryRunServer, nil
			}).
				DependsOn(local.OptionalDependency()).
				DependsOn(dryRun.OptionalDependency())
		},
		Run: runSetSubject,
	}
	return cmd
}
