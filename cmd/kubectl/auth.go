package main

import (
	"context"
	"io"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/kubectl/pkg/cmd/util"
)

type authCanITarget struct {
	NonResourceURL string
	Resource       string
	ResourceName   string
}

type authCanIConfig struct {
	Client clientConfig
	Stdout io.Writer
	Stderr io.Writer

	Verb          string
	Target        authCanITarget
	AllNamespaces bool
	List          bool
	NoHeaders     bool
	Quiet         bool
	Subresource   string
}

type authReconcileConfig struct {
	Client clientConfig
	Stdout io.Writer
	Stderr io.Writer

	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	Filename                 []string
	Kustomize                string
	Output                   string
	Recursive                bool
	RemoveExtraPermissions   bool
	RemoveExtraSubjects      bool
	ShowManagedFields        bool
	Template                 string
}

type authWhoAmIConfig struct {
	Client clientConfig
	Stdout io.Writer

	AllowMissingTemplateKeys bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	ShowManagedFields        bool
	Template                 string
}

var authCanITargetLanguage = cli.ValueCodec[authCanITarget]{
	Name: "kubectl-auth-target",
	Parse: func(value string) (authCanITarget, error) {
		if strings.HasPrefix(value, "/") {
			target := authCanITarget{
				NonResourceURL: value,
			}
			return target, nil
		}
		resource, resourceName, _ := strings.Cut(value, "/")
		target := authCanITarget{
			Resource:     resource,
			ResourceName: resourceName,
		}
		return target, nil
	},
	Format: func(value authCanITarget) string {
		if value.NonResourceURL != "" {
			return value.NonResourceURL
		}
		if value.ResourceName != "" {
			return value.Resource + "/" + value.ResourceName
		}
		return value.Resource
	},
	Snapshot: cli.Value.Assign[authCanITarget](),
}

var authReconcileDryRunLanguage = cli.ValueCodec[util.DryRunStrategy]{
	Name:  "kubectl-dry-run-strategy",
	Parse: dryRunStrategy,
	Format: func(value util.DryRunStrategy) string {
		switch value {
		case util.DryRunClient:
			return "client"
		case util.DryRunServer:
			return "server"
		default:
			return "none"
		}
	},
	Snapshot: cli.Value.Assign[util.DryRunStrategy](),
}

func authCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "auth",
		Description: cli.Description{
			Short: "Inspect authorization",
			Long:  "Inspect authorization.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			canI := authCanICommand(client)
			reconcile := authReconcileCommand(client)
			whoAmI := authWhoAmICommand(client)
			cb.Subcommands.Command(canI)
			cb.Subcommands.Command(reconcile)
			cb.Subcommands.Command(whoAmI)
		},
	}
	return cmd
}

func authCanICommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[authCanIConfig] {
	cmd := cli.Command[authCanIConfig]{
		Name: "can-i",
		Description: cli.Description{
			Short: "Check whether an action is allowed",
			Long:  "Check whether an action is allowed.\n\n VERB is a logical Kubernetes API verb like 'get', 'list', 'watch', 'delete', etc. TYPE is a Kubernetes resource. Shortcuts and groups will be resolved. NONRESOURCEURL is a partial URL that starts with \"/\". NAME is the name of a particular Kubernetes resource. This command pairs nicely with impersonation. See --as global flag.",
		},
		Examples: []cli.Example{
			{
				Description: "Check to see if I can create pods in any namespace",
				CommandLine: "kubectl auth can-i create pods --all-namespaces",
			},
			{
				Description: "Check to see if I can list deployments in my current namespace",
				CommandLine: "kubectl auth can-i list deployments.apps",
			},
			{
				Description: "Check to see if service account \"foo\" of namespace \"dev\" can list pods in the namespace \"prod\"\nYou must be allowed to use impersonation for the global option \"--as\"",
				CommandLine: "kubectl auth can-i list pods --as=system:serviceaccount:dev:foo -n prod",
			},
			{
				Description: "Check to see if I can do everything in my current namespace (\"*\" means all)",
				CommandLine: "kubectl auth can-i '*' '*'",
			},
			{
				Description: "Check to see if I can get the job named \"bar\" in namespace \"foo\"",
				CommandLine: "kubectl auth can-i list jobs.batch/bar -n foo",
			},
			{
				Description: "Check to see if I can read pod logs",
				CommandLine: "kubectl auth can-i get pods --subresource=log",
			},
			{
				Description: "Check to see if I can access the URL /logs/",
				CommandLine: "kubectl auth can-i get /logs/",
			},
			{
				Description: "Check to see if I can approve certificates.k8s.io",
				CommandLine: "kubectl auth can-i approve certificates.k8s.io",
			},
			{
				Description: "List all allowed actions in namespace \"foo\"",
				CommandLine: "kubectl auth can-i --list --namespace=foo",
			},
		},
		Declare: func(cb *cli.CommandBuilder[authCanIConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			verb := cb.Values.String(&cb.Config.Verb)
			target := cb.Values.Value(&cb.Config.Target, authCanITargetLanguage)
			cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[authCanIConfig]) {})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[authCanIConfig]) {
				form.ValueFor(verb, "VERB")
				form.ValueFor(target, "TYPE | TYPE/NAME | NONRESOURCEURL")
			})

			allNamespaces := cb.Values.Bool(&cb.Config.AllNamespaces)
			allNamespacesFlag := cb.Flags.ValueFor(allNamespaces).
				Description("If true, check the specified action in all namespaces.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			list := cb.Values.Bool(&cb.Config.List)
			cb.Flags.ValueFor(list).
				Description("If true, prints all allowed actions.").
				Long("list").
				ImplicitValue(true)

			noHeaders := cb.Values.Bool(&cb.Config.NoHeaders)
			cb.Flags.ValueFor(noHeaders).
				Description("If true, prints allowed actions without headers").
				Long("no-headers").
				ImplicitValue(true)

			quiet := cb.Values.Bool(&cb.Config.Quiet)
			quietFlag := cb.Flags.ValueFor(quiet).
				Description("If true, suppress output and just return the exit code.")
			quietFlag.Long("quiet").ImplicitValue(true)
			quietFlag.Short('q').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			subresource := cb.Values.String(&cb.Config.Subresource)
			cb.Flags.ValueFor(subresource).
				Description("SubResource such as pod/log or deployment/scale").
				Long("subresource")

			listing := cb.When.Equal(list, true)
			cb.Rule.When(listing).Forbid(verb)
			cb.Rule.When(listing).Forbid(target)
			cb.Rule.Unless(listing).Require(verb)
			cb.Rule.Unless(listing).Require(target)

			cb.Rule.When(cb.When.Equal(quiet, true)).NotEqual(list, true)
			cb.Rule.When(cb.When.Equal(allNamespaces, true)).NotEqual(list, true)
			cb.Rule.When(cb.When.NotEqual(subresource, "")).NotEqual(list, true)
			withoutHeaders := cb.When.Equal(noHeaders, true)
			cb.Rule.When(withoutHeaders).Require(list)
			cb.Rule.When(withoutHeaders).Equal(list, true)
			cb.Rule.Check("auth-can-i-nonresource-subresource", func(_ context.Context, cfg *authCanIConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Target.NonResourceURL == "" || cfg.Subresource == "", nil
			}).Subject(target).Subject(subresource)
		},
		Run: runAuthCanI,
	}
	return cmd
}

func authReconcileCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[authReconcileConfig] {
	cmd := cli.Command[authReconcileConfig]{
		Name: "reconcile",
		Description: cli.Description{
			Short: "Reconciles rules for RBAC role, role binding, cluster role, and cluster role binding objects",
			Long:  "Reconciles rules for RBAC role, role binding, cluster role, and cluster role binding objects.\n\n Missing objects are created, and the containing namespace is created for namespaced objects, if required.\n\n Existing roles are updated to include the permissions in the input objects, and remove extra permissions if --remove-extra-permissions is specified.\n\n Existing bindings are updated to include the subjects in the input objects, and remove extra subjects if --remove-extra-subjects is specified.\n\n This is preferred to 'apply' for RBAC resources so that semantically-aware merging of rules and subjects is done.",
		},
		Examples: []cli.Example{
			{
				Description: "Reconcile RBAC resources from a file",
				CommandLine: "kubectl auth reconcile -f my-rbac-rules.yaml",
			},
		},
		Declare: func(cb *cli.CommandBuilder[authReconcileConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			allowMissingTemplateKeys := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissingTemplateKeys, true)
			cb.Flags.ValueFor(allowMissingTemplateKeys).
				Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").
				Long("allow-missing-template-keys").
				ImplicitValue(true)

			dryRun := cb.Values.Value(&cb.Config.DryRun, authReconcileDryRunLanguage)
			cb.Values.Default(dryRun, util.DryRunNone)
			cb.Flags.ValueFor(dryRun).
				Description("Must be \"none\" or \"client\". If client strategy, only print the object that would be sent, without sending it.").
				Long("dry-run").
				ImplicitValue(util.DryRunClient)
			cb.Rule.NotEqual(dryRun, util.DryRunServer)

			filename := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filename).
				DelimitedBy(",").
				Description("Filename, directory, or URL to files identifying the resource to reconcile.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize).
				Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			output := cb.Values.String(&cb.Config.Output)
			outputFlag := cb.Flags.ValueFor(output).
				Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recursive := cb.Values.Bool(&cb.Config.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive).
				Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.RemoveExtraPermissions)).
				Description("If true, removes extra permissions added to roles").
				Long("remove-extra-permissions").
				ImplicitValue(true)

			cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.RemoveExtraSubjects)).
				Description("If true, removes extra subjects added to rolebindings").
				Long("remove-extra-subjects").
				ImplicitValue(true)

			cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields)).
				Description("If true, keep the managedFields when printing objects in JSON or YAML format.").
				Long("show-managed-fields").
				ImplicitValue(true)

			cb.Flags.String(&cb.Config.Template).
				Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").
				Long("template")

			cb.Rule.ExactlyOne(filename, kustomize)
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
		},
		Run: runAuthReconcile,
	}
	return cmd
}

func authWhoAmICommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[authWhoAmIConfig] {
	cmd := cli.Command[authWhoAmIConfig]{
		Name: "whoami",
		Description: cli.Description{
			Short: "Experimental: Check self subject attributes",
			Long:  "Experimental: Check who you are and your attributes (groups, extra).\n\n        This command is helpful to get yourself aware of the current user attributes,\n        especially when dynamic authentication, e.g., token webhook, auth proxy, or OIDC provider,\n        is enabled in the Kubernetes cluster.",
		},
		Examples: []cli.Example{
			{
				Description: "Get your subject attributes",
				CommandLine: "kubectl auth whoami",
			},
			{
				Description: "Get your subject attributes in JSON format",
				CommandLine: "kubectl auth whoami -o json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[authWhoAmIConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Stdout)

			allowMissing := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissing, true)
			cb.Flags.ValueFor(allowMissing).
				Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").
				Long("allow-missing-template-keys").
				ImplicitValue(true)

			output := cb.Values.String(&cb.Config.Output)
			outputFlag := cb.Flags.ValueFor(output).
				Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')
			cb.Access.ValueState(output, &cb.Config.OutputState)

			cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields)).
				Description("If true, keep the managedFields when printing objects in JSON or YAML format.").
				Long("show-managed-fields").
				ImplicitValue(true)

			cb.Flags.String(&cb.Config.Template).
				Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").
				Long("template")
		},
		Run: runAuthWhoAmI,
	}
	return cmd
}
