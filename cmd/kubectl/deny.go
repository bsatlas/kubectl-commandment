package main

import (
	"github.com/abstractionsinc/commandment/cli"
)

func certificateDenyCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[certificateConditionConfig] {
	cmd := cli.Command[certificateConditionConfig]{
		Name: "deny",
		Description: cli.Description{
			Short: "Deny a certificate signing request",
			Long:  "Deny a certificate signing request.\n\n kubectl certificate deny allows a cluster admin to deny a certificate signing request (CSR). This action tells a certificate signing controller to not to issue a certificate to the requester.",
		},
		Examples: []cli.Example{
			{
				Description: "Deny CSR 'csr-sqgzp'",
				CommandLine: "kubectl certificate deny csr-sqgzp",
			},
		},
		Declare: func(cb *cli.CommandBuilder[certificateConditionConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[certificateConditionConfig]) {
				form.Sequence(arguments, "NAME", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").Long("allow-missing-template-keys").ImplicitValue(true)
			cb.Values.Default(allowMissingTemplateKeysFlag, true)

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",").Description("Filename, directory, or URL to files identifying the resource to update").Long("filename")
			filenameFlag.Short('f')

			forceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Force))
			forceFlag.Description("Update the CSR even if it is already denied.").Long("force").ImplicitValue(true)

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.").Long("kustomize")
			kustomizeFlag.Short('k')

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).").Long("output")
			outputFlag.Short('o')

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag.Description("Process the directory used in -f, --filename recursively.").Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.").Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file.").Long("template")
		},
		Run: runCertificateDeny,
	}
	return cmd
}
