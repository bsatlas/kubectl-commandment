package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type waitConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	All                      bool
	AllNamespaces            bool
	AllowMissingTemplateKeys bool
	FieldSelector            string
	Filename                 []string
	For                      []string
	Local                    bool
	Output                   string
	Recursive                bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Timeout                  time.Duration
}

func waitCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[waitConfig] {
	cmd := cli.Command[waitConfig]{
		Name: "wait",
		Description: cli.Description{
			Short: "Wait for a specific condition on one or many resources",
			Long:  "Wait for a specific condition on one or many resources.\n\n The command takes multiple resources and waits until the specified condition is seen in the Status field of every given resource.\n\n Alternatively, the command can wait for the given set of resources to be created or deleted by providing the \"create\" or \"delete\" keyword as the value to the --for flag.\n\n A successful message will be printed to stdout indicating when the specified condition has been met. You can use -o option to change to output destination.",
		},
		Examples: []cli.Example{
			{
				Description: "Wait for the pod \"busybox1\" to contain the status condition of type \"Ready\"",
				CommandLine: "kubectl wait --for=condition=Ready pod/busybox1",
			},
			{
				Description: "The default value of status condition is true; you can wait for other targets after an equal delimiter (compared after Unicode simple case folding, which is a more general form of case-insensitivity)",
				CommandLine: "kubectl wait --for=condition=Ready=false pod/busybox1",
			},
			{
				Description: "Wait for the pod \"busybox1\" to contain the status phase to be \"Running\"",
				CommandLine: "kubectl wait --for=jsonpath='{.status.phase}'=Running pod/busybox1",
			},
			{
				Description: "Wait for pod \"busybox1\" to be Ready",
				CommandLine: "kubectl wait --for='jsonpath={.status.conditions[?(@.type==\"Ready\")].status}=True' pod/busybox1",
			},
			{
				Description: "Wait for the service \"loadbalancer\" to have ingress",
				CommandLine: "kubectl wait --for=jsonpath='{.status.loadBalancer.ingress}' service/loadbalancer",
			},
			{
				Description: "Wait for the secret \"busybox1\" to be created, with a timeout of 30s",
				CommandLine: "kubectl create secret generic busybox1\n  kubectl wait --for=create secret/busybox1 --timeout=30s",
			},
			{
				Description: "Wait for the pod \"busybox1\" to be deleted, with a timeout of 60s, after having issued the \"delete\" command",
				CommandLine: "kubectl delete pod/busybox1\n  kubectl wait --for=delete pod/busybox1 --timeout=60s",
			},
			{
				Description: "Wait for pod \"busybox1\" to be created AND reach the \"Ready\" status condition",
				CommandLine: "kubectl wait --for=condition=Ready --for=create pod/busybox1",
			},
			{
				Description: "Wait for pod \"busybox1\" to reach the \"Ready\" status OR for its containers to report a \"False\" readiness state",
				CommandLine: "until kubectl wait pod/busybox1 --for=condition=Ready --timeout=1s 2>/dev/null || \\\n  kubectl wait pod/busybox1 --for=condition=ContainersReady=False --timeout=1s 2>/dev/null; \\\n  do echo \"Checking conditions...\"; sleep 1; done",
			},
		},
		Declare: func(cb *cli.CommandBuilder[waitConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[waitConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore())
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag = allFlag.Description("Select all resources in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespacesFlag = allNamespacesFlag.Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			fieldSelectorFlag := cb.Flags.String(&cb.Config.FieldSelector)
			fieldSelectorFlag = fieldSelectorFlag.Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelectorFlag.Long("field-selector")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("identifying the resource.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			forValue := cb.Values.Sequence(&cb.Config.For, cli.Value.String())
			forFlag := cb.Flags.Sequence(forValue)
			forFlag = forFlag.Description("The condition to wait on: [create|delete|condition=condition-name[=condition-value]|jsonpath='{JSONPath expression}'=[JSONPath value]]. The default condition-value is true. Condition values are compared after Unicode simple case folding, which is a more general form of case-insensitivity. Multiple conditions are supported and AND'ed to each other in a sequential order. If --for=create is passed, it is always waited first.")
			forFlag.Long("for")

			localFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Local))
			localFlag = localFlag.Description("If true, annotation will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			cb.Values.Default(recursiveFlag, true)
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag = timeoutFlag.Description("The length of time to wait before giving up. Zero means check once and don't wait, negative means wait for a week.")
			cb.Values.Default(timeoutFlag, 30*time.Second)
			timeoutFlag.Long("timeout")
		},
		Run: runWait,
	}
	return cmd
}
