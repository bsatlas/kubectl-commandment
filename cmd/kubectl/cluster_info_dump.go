package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type clusterInfoDumpConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	AllNamespaces            bool
	AllowMissingTemplateKeys bool
	Namespaces               []string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	OutputDirectory          string
	PodRunningTimeout        time.Duration
	ShowManagedFields        bool
	Template                 string
	IgnoredArgs              []string
}

func clusterInfoDumpCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[clusterInfoDumpConfig] {
	cmd := cli.Command[clusterInfoDumpConfig]{
		Name: "dump",
		Description: cli.Description{
			Short: "Dump relevant information for debugging and diagnosis",
			Long:  "Dump cluster information out suitable for debugging and diagnosing cluster problems.  By default, dumps everything to stdout. You can optionally specify a directory with --output-directory.  If you specify a directory, Kubernetes will build a set of files in that directory.  By default, only dumps things in the current namespace and 'kube-system' namespace, but you can switch to a different namespace with the --namespaces flag, or specify --all-namespaces to dump all namespaces.\n\n The command also dumps the logs of all of the pods in the cluster; these logs are dumped into different directories based on namespace and pod name.",
		},
		Examples: []cli.Example{
			{
				Description: "Dump current cluster state to stdout",
				CommandLine: "kubectl cluster-info dump",
			},
			{
				Description: "Dump current cluster state to /path/to/cluster-state",
				CommandLine: "kubectl cluster-info dump --output-directory=/path/to/cluster-state",
			},
			{
				Description: "Dump all namespaces to stdout",
				CommandLine: "kubectl cluster-info dump --all-namespaces",
			},
			{
				Description: "Dump a set of namespaces to /path/to/cluster-state",
				CommandLine: "kubectl cluster-info dump --namespaces default,kube-system --output-directory=/path/to/cluster-state",
			},
		},
		Declare: func(cb *cli.CommandBuilder[clusterInfoDumpConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces)).Description("If true, dump all namespaces.  If true, --namespaces is ignored.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)).Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			namespacesValue := cb.Values.Sequence(&cb.Config.Namespaces, cli.Value.String())
			namespacesFlag := cb.Flags.Sequence(namespacesValue).
				DelimitedBy(",").
				Description("A comma separated list of namespaces to dump.")
			namespacesFlag.Long("namespaces")

			output := cb.Values.Value(&cb.Config.Output, printOutputLanguage)
			cb.Values.Default(output, "json")
			outputFlag := cb.Flags.ValueFor(output).Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			cb.Access.ValueState(outputFlag, &cb.Config.OutputState)

			outputDirectoryFlag := cb.Flags.String(&cb.Config.OutputDirectory).Description("Where to output the files.  If empty or '-' uses stdout, otherwise creates a directory hierarchy in that directory")
			outputDirectoryFlag.Long("output-directory")

			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage()).Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, 20*time.Second)
			podRunningTimeoutFlag.Long("pod-running-timeout")
			cb.Rule.GreaterThan(podRunningTimeoutFlag.Destination(), time.Duration(0))

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields)).Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template).Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").CompleteWith(cli.CompletionNativeFiles)
			templateFlag.Long("template")

			declarePrintOutputTemplateRule(cb, outputFlag, templateFlag)

			ignoredArgs := cb.Values.Sequence(&cb.Config.IgnoredArgs, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[clusterInfoDumpConfig]) {
				form.Sequence(ignoredArgs, "ARG", cli.Count.ZeroOrMore()).Hidden()
			})
		},
		Run: runClusterInfoDump,
	}
	return cmd
}
