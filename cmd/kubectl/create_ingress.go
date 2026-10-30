package main

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var ingressRulePattern = regexp.MustCompile(`^(?P<host>[\w\*\-\.]*)(?P<path>/[^,=\n]*)=(?P<svcname>[\w\-]+):(?P<svcport>[\w\-]+)(,(?P<istls>tls)=?(?P<secretname>[\w\-]+)?)?$`)

var ingressRuleLanguage = cli.ValueCodec[string]{
	Name: "ingress-rule",
	Parse: func(value string) (string, error) {
		if !ingressRulePattern.MatchString(value) {
			return "", cli.Rejection.Message("ingress-rule", "expected host/path=service:port[,tls[=secret]]", cli.Rejection.At(0, 0, len(value)))
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

var ingressAnnotationLanguage = cli.ValueCodec[string]{
	Name: "ingress-annotation",
	Parse: func(value string) (string, error) {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", cli.Rejection.Message("ingress-annotation", "expected key=value", cli.Rejection.At(0, 0, len(value)))
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

var ingressBackendLanguage = cli.ValueCodec[string]{
	Name: "ingress-backend",
	Parse: func(value string) (string, error) {
		parts := strings.Split(value, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", cli.Rejection.Message("ingress-backend", "expected service:port", cli.Rejection.At(0, 0, len(value)))
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

type createIngressConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	Annotations              []string
	Class                    string
	DefaultBackend           string
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Rules                    []string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createIngressCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createIngressConfig] {
	cmd := cli.Command[createIngressConfig]{
		Name:    "ingress",
		Aliases: cli.Aliases("ing"),
		Description: cli.Description{
			Short: "Create an ingress with the specified name",
			Long:  "Create an ingress with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a single ingress called 'simple' that directs requests to foo.com/bar to svc\nsvc1:8080 with a TLS secret \"my-cert\"",
				CommandLine: "kubectl create ingress simple --rule=\"foo.com/bar=svc1:8080,tls=my-cert\"",
			},
			{
				Description: "Create a catch all ingress of \"/path\" pointing to service svc:port and Ingress Class as \"otheringress\"",
				CommandLine: "kubectl create ingress catch-all --class=otheringress --rule=\"/path=svc:port\"",
			},
			{
				Description: "Create an ingress with two annotations: ingress.annotation1 and ingress.annotations2",
				CommandLine: "kubectl create ingress annotated --class=default --rule=\"foo.com/bar=svc:port\" \\\n  --annotation ingress.annotation1=foo \\\n  --annotation ingress.annotation2=bla",
			},
			{
				Description: "Create an ingress with the same host and multiple paths",
				CommandLine: "kubectl create ingress multipath --class=default \\\n  --rule=\"foo.com/=svc:port\" \\\n  --rule=\"foo.com/admin/=svcadmin:portadmin\"",
			},
			{
				Description: "Create an ingress with multiple hosts and the pathType as Prefix",
				CommandLine: "kubectl create ingress ingress1 --class=default \\\n  --rule=\"foo.com/path*=svc:8080\" \\\n  --rule=\"bar.com/admin*=svc2:http\"",
			},
			{
				Description: "Create an ingress with TLS enabled using the default ingress certificate and different path types",
				CommandLine: "kubectl create ingress ingtls --class=default \\\n  --rule=\"foo.com/=svc:https,tls\" \\\n  --rule=\"foo.com/path/subpath*=othersvc:8080\"",
			},
			{
				Description: "Create an ingress with TLS enabled using a specific secret and pathType as Prefix",
				CommandLine: "kubectl create ingress ingsecret --class=default \\\n  --rule=\"foo.com/*=svc:8080,tls=secret1\"",
			},
			{
				Description: "Create an ingress with a default backend",
				CommandLine: "kubectl create ingress ingdefault --class=default \\\n  --default-backend=defaultsvc:http \\\n  --rule=\"foo.com/*=svc:8080,tls=secret1\"",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createIngressConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createIngressConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createIngressConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			annotationValue := cb.Values.Sequence(&cb.Config.Annotations, ingressAnnotationLanguage)
			annotationFlag := cb.Flags.Sequence(annotationValue)
			annotationFlag.Description("Annotation to insert in the ingress object, in the format annotation=value")
			annotationFlag.Long("annotation")

			classFlag := cb.Flags.String(&cb.Config.Class)
			cb.Values.ResolveWith(classFlag, cli.Value.KeepLast())
			classFlag.Description("Ingress Class to be used")
			classFlag.Long("class")

			defaultBackend := cb.Values.Value(&cb.Config.DefaultBackend, ingressBackendLanguage)
			defaultBackendFlag := cb.Flags.ValueFor(defaultBackend)
			cb.Values.ResolveWith(defaultBackendFlag, cli.Value.KeepLast())
			defaultBackendFlag.Description("Default service for backend, in format of svcname:port")
			defaultBackendFlag.Long("default-backend")

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

			ruleValue := cb.Values.Sequence(&cb.Config.Rules, ingressRuleLanguage)
			ruleFlag := cb.Flags.Sequence(ruleValue)
			ruleFlag.Description("Rule in format host/path=service:port[,tls=secretname]. Paths containing the leading character '*' are considered pathType=Prefix. tls argument is optional.")
			ruleFlag.Long("rule")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Check("kubectl-create-ingress-backend-or-rule", func(ctx context.Context, cfg *createIngressConfig, _ cli.Dependencies) (bool, error) {
				return cfg.DefaultBackend != "" || len(cfg.Rules) > 0, nil
			}).
				DependsOn(defaultBackend.OptionalDependency()).
				DependsOn(ruleValue.OptionalDependency())
		},
		Run: runCreateIngress,
	}
	return cmd
}

func runCreateIngress(ctx context.Context, cfg *createIngressConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewCreateIngressOptions(cfg.Streams)
	options.Name = cfg.Name
	options.IngressClass = cfg.Class
	options.Rules = slices.Clone(cfg.Rules)
	options.Annotations = slices.Clone(cfg.Annotations)
	options.DefaultBackend = cfg.DefaultBackend
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.NetworkingV1()
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
