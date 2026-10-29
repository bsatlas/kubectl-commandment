package main

import (
	"context"
	"io"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
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
