package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type createIngressConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	Annotations              []string
	Class                    string
	DefaultBackend           string
	DryRun                   string
	FieldManager             string
	Output                   string
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

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			annotationValue := cb.Values.Sequence(&cb.Config.Annotations, cli.Value.String())
			annotationFlag := cb.Flags.Sequence(annotationValue)
			annotationFlag = annotationFlag.Description("Annotation to insert in the ingress object, in the format annotation=value")
			annotationFlag.Long("annotation")

			classFlag := cb.Flags.String(&cb.Config.Class)
			classFlag = classFlag.Description("Ingress Class to be used")
			classFlag.Long("class")

			defaultBackendFlag := cb.Flags.String(&cb.Config.DefaultBackend)
			defaultBackendFlag = defaultBackendFlag.Description("Default service for backend, in format of svcname:port")
			defaultBackendFlag.Long("default-backend")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			ruleValue := cb.Values.Sequence(&cb.Config.Rules, cli.Value.String())
			ruleFlag := cb.Flags.Sequence(ruleValue)
			ruleFlag = ruleFlag.Description("Rule in format host/path=service:port[,tls=secretname]. Paths containing the leading character '*' are considered pathType=Prefix. tls argument is optional.")
			ruleFlag.Long("rule")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")
		},
		Run: runCreateIngress,
	}
	return cmd
}
