package main

import (
	"context"
	"regexp"

	"github.com/abstractionsinc/commandment/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	policyv1 "k8s.io/client-go/kubernetes/typed/policy/v1"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var podDisruptionBudgetSelectorLanguage = cli.ValueCodec[string]{
	Name: "label-selector",
	Parse: func(value string) (string, error) {
		if _, err := metav1.ParseToLabelSelector(value); err != nil {
			return "", cli.Rejection.Message("label-selector", err.Error(), cli.Rejection.At(0, 0, len(value)))
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

var podDisruptionBudgetAvailabilityPattern = regexp.MustCompile(`^[0-9]+%?$`)

var podDisruptionBudgetAvailabilityLanguage = cli.ValueCodec[string]{
	Name: "availability",
	Parse: func(value string) (string, error) {
		if !podDisruptionBudgetAvailabilityPattern.MatchString(value) {
			return "", cli.Rejection.Message("availability", "expected a non-negative integer or percentage", cli.Rejection.At(0, 0, len(value)))
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

type createPodDisruptionBudgetConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	MaxUnavailable           string
	MinAvailable             string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	SaveConfig               bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createPodDisruptionBudgetCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createPodDisruptionBudgetConfig] {
	cmd := cli.Command[createPodDisruptionBudgetConfig]{
		Name:    "poddisruptionbudget",
		Aliases: cli.Aliases("pdb"),
		Description: cli.Description{
			Short: "Create a pod disruption budget with the specified name",
			Long:  "Create a pod disruption budget with the specified name, selector, and desired minimum available pods.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a pod disruption budget named my-pdb that will select all pods with the app=rails label\nand require at least one of them being available at any point in time",
				CommandLine: "kubectl create poddisruptionbudget my-pdb --selector=app=rails --min-available=1",
			},
			{
				Description: "Create a pod disruption budget named my-pdb that will select all pods with the app=nginx label\nand require at least half of the pods selected to be available at any point in time",
				CommandLine: "kubectl create pdb my-pdb --selector=app=nginx --min-available=50%",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createPodDisruptionBudgetConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createPodDisruptionBudgetConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createPodDisruptionBudgetConfig]) {
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

			maxUnavailable := cb.Values.Value(&cb.Config.MaxUnavailable, podDisruptionBudgetAvailabilityLanguage)
			maxUnavailableFlag := cb.Flags.ValueFor(maxUnavailable)
			cb.Values.ResolveWith(maxUnavailableFlag, cli.Value.KeepLast())
			maxUnavailableFlag.Description("The maximum number or percentage of unavailable pods this budget requires.")
			maxUnavailableFlag.Long("max-unavailable")

			minAvailable := cb.Values.Value(&cb.Config.MinAvailable, podDisruptionBudgetAvailabilityLanguage)
			minAvailableFlag := cb.Flags.ValueFor(minAvailable)
			cb.Values.ResolveWith(minAvailableFlag, cli.Value.KeepLast())
			minAvailableFlag.Description("The minimum number or percentage of available pods this budget requires.")
			minAvailableFlag.Long("min-available")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			selectorFlag := cb.Flags.Value(&cb.Config.Selector, podDisruptionBudgetSelectorLanguage)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("A label selector to use for this budget. Only equality-based selector requirements are supported.")
			selectorFlag.Long("selector")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Require(selectorFlag.Destination())
			cb.Rule.NotEmpty(selectorFlag.Destination())
			cb.Rule.ExactlyOne(minAvailable, maxUnavailable)
		},
		Run: runCreatePodDisruptionBudget,
	}
	return cmd
}

func runCreatePodDisruptionBudget(ctx context.Context, cfg *createPodDisruptionBudgetConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewPodDisruptionBudgetOpts(cfg.Streams)
	options.Name = cfg.Name
	options.MinAvailable = cfg.MinAvailable
	options.MaxUnavailable = cfg.MaxUnavailable
	options.Selector = cfg.Selector
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = policyv1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
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
