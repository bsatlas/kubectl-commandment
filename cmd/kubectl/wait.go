package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/util/jsonpath"
	kubectlget "k8s.io/kubectl/pkg/cmd/get"
	kubectlwait "k8s.io/kubectl/pkg/cmd/wait"
)

var waitConditionLanguage = cli.ValueCodec[string]{
	Name: "kubectl-wait-condition",
	Parse: func(value string) (string, error) {
		if err := validateWaitCondition(value); err != nil {
			return "", cli.Rejection.Message(
				"wait-condition",
				err.Error(),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

func validateWaitCondition(condition string) error {
	switch lowercase := strings.ToLower(condition); {
	case lowercase == "delete", lowercase == "create":
		return nil
	case strings.HasPrefix(condition, "condition="):
		return nil
	case strings.HasPrefix(condition, "jsonpath="):
		expression, err := waitJSONPathExpression(strings.TrimPrefix(condition, "jsonpath="))
		if err != nil {
			return err
		}
		parser := jsonpath.New("wait").AllowMissingKeys(true)
		return parser.Parse(expression)
	default:
		return fmt.Errorf("unrecognized condition: %q", condition)
	}
}

func waitJSONPathExpression(input string) (string, error) {
	parts := splitWaitJSONPathInput(input)
	if len(parts) < 1 || len(parts) > 2 {
		return "", fmt.Errorf("jsonpath wait format must be --for=jsonpath='{.status.readyReplicas}'=3 or --for=jsonpath='{.status.readyReplicas}'")
	}
	expression, err := kubectlget.RelaxedJSONPathExpression(parts[0])
	if err != nil {
		return "", err
	}
	if expression == "" {
		return "", errors.New("jsonpath expression cannot be empty")
	}
	if len(parts) == 2 && strings.Trim(parts[1], `'"`) == "" {
		return "", errors.New("jsonpath wait has to have a value after equal sign, like --for=jsonpath='{.status.readyReplicas}'=3")
	}
	return expression, nil
}

func splitWaitJSONPathInput(input string) []string {
	var output []string
	var element strings.Builder
	for index := 0; index < len(input); index++ {
		if input[index] == '=' {
			if index < len(input)-1 && input[index+1] == '=' {
				element.WriteString("==")
				index++
				continue
			}
			output = append(output, element.String())
			element.Reset()
			continue
		}
		element.WriteByte(input[index])
	}
	return append(output, element.String())
}

type waitConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	All                      bool
	AllNamespaces            bool
	AllowMissingTemplateKeys bool
	FieldSelector            string
	Filename                 filenameOptions
	For                      []string
	Local                    bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Timeout                  time.Duration
	RequestedTimeout         time.Duration
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

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("Select all resources in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			cb.Values.ResolveWith(allNamespacesFlag, cli.Value.KeepLast())
			allNamespacesFlag.Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			declarePrintFlags(cb, &cb.Config.Output, &cb.Config.OutputState, &cb.Config.Template, &cb.Config.AllowMissingTemplateKeys, &cb.Config.ShowManagedFields)

			fieldSelector := cb.Values.String(&cb.Config.FieldSelector)
			fieldSelectorFlag := cb.Flags.ValueFor(fieldSelector)
			cb.Values.ResolveWith(fieldSelectorFlag, cli.Value.KeepLast())
			fieldSelectorFlag.Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelectorFlag.Long("field-selector")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("identifying the resource.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			forValue := cb.Values.Sequence(&cb.Config.For, waitConditionLanguage)
			forFlag := cb.Flags.Sequence(forValue)
			forFlag.Description("The condition to wait on: [create|delete|condition=condition-name[=condition-value]|jsonpath='{JSONPath expression}'=[JSONPath value]]. The default condition-value is true. Condition values are compared after Unicode simple case folding, which is a more general form of case-insensitivity. Multiple conditions are supported and AND'ed to each other in a sequential order. If --for=create is passed, it is always waited first.")
			forFlag.Long("for")
			cb.Rule.Require(forValue)

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			cb.Values.ResolveWith(localFlag, cli.Value.KeepLast())
			localFlag.Description("If true, annotation will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Filename.Recursive))
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			cb.Values.Default(recursiveFlag, true)
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			requestedTimeout := cb.Values.Value(&cb.Config.RequestedTimeout, durationLanguage())
			cb.Values.Default(requestedTimeout, 30*time.Second)
			timeoutFlag := cb.Flags.ValueFor(requestedTimeout)
			cb.Values.ResolveWith(timeoutFlag, cli.Value.KeepLast())
			timeoutFlag.Description("The length of time to wait before giving up. Zero means check once and don't wait, negative means wait for a week.")
			timeoutFlag.Long("timeout")

			cb.Rule.Check("kubectl-wait-source", func(_ context.Context, cfg *waitConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Arguments) > 0 || len(cfg.Filename.Filenames()) > 0, nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency())
			cb.Rule.When(cb.When.Equal(local, true)).Forbid(arguments)
			cb.Values.Derive(&cb.Config.Timeout, cli.Value.ComputedBy(func(_ context.Context, _ *waitConfig, dependencies cli.Dependencies) (time.Duration, error) {
				timeout, ok := dependencies.Get(requestedTimeout)
				if !ok {
					return 0, fmt.Errorf("wait timeout dependency is unavailable")
				}
				if timeout < 0 {
					return 168 * time.Hour, nil
				}
				return timeout, nil
			}).From(requestedTimeout))
		},
		Run: func(ctx context.Context, cfg *waitConfig) error {
			flags := kubectlwait.NewWaitFlags(restClientGetter(cfg.Client), cfg.Streams)
			flags.Timeout = cfg.Timeout
			flags.ForCondition = slices.Clone(cfg.For)
			flags.ResourceBuilderFlags.
				WithFile(cfg.Filename.Recursive, nil, cfg.Filename.Filenames()...).
				WithLabelSelector(cfg.Selector).
				WithFieldSelector(cfg.FieldSelector).
				WithAllNamespaces(cfg.AllNamespaces).
				WithAll(cfg.All).
				WithLocal(cfg.Local)
			configurePrintFlagsWithOutputState(flags.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
			options, err := flags.ToOptions(cfg.Arguments)
			if err != nil {
				return err
			}
			return options.RunWaitContext(ctx)
		},
	}
	return cmd
}
