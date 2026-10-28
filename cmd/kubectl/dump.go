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
	OutputDirectory          string
	PodRunningTimeout        time.Duration
	ShowManagedFields        bool
	Template                 string
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

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespacesFlag = allNamespacesFlag.Description("If true, dump all namespaces.  If true, --namespaces is ignored.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			namespacesValue := cb.Values.Sequence(&cb.Config.Namespaces, cli.Value.String())
			namespacesFlag := cb.Flags.Sequence(namespacesValue)
			namespacesFlag = namespacesFlag.DelimitedBy(",")
			namespacesFlag = namespacesFlag.Description("A comma separated list of namespaces to dump.")
			namespacesFlag.Long("namespaces")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			cb.Values.Default(outputFlag, "json")
			outputFlag.Long("output")
			outputFlag.Short('o')

			outputDirectoryFlag := cb.Flags.String(&cb.Config.OutputDirectory)
			outputDirectoryFlag = outputDirectoryFlag.Description("Where to output the files.  If empty or '-' uses stdout, otherwise creates a directory hierarchy in that directory")
			outputDirectoryFlag.Long("output-directory")

			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			podRunningTimeoutFlag = podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, 20*time.Second)
			podRunningTimeoutFlag.Long("pod-running-timeout")

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")
		},
		Run: runClusterInfoDump,
	}
	return cmd
}
