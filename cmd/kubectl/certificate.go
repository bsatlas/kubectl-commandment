package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type certificateConditionConfig struct {
	Client clientConfig
	Stdin  io.Reader
	Stdout io.Writer

	Names                    []string
	AllowMissingTemplateKeys bool
	Filename                 []string
	Force                    bool
	Kustomize                string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Recursive                bool
	ShowManagedFields        bool
	Template                 string
}

func certificateCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "certificate",
		Description: cli.Description{
			Short: "Modify certificate resources",
			Long:  "Modify certificate resources.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			approve := certificateApproveCommand(client)
			deny := certificateDenyCommand(client)
			cb.Subcommands.Command(approve)
			cb.Subcommands.Command(deny)
		},
	}
	return cmd
}

func declareCertificateConditionCommand(cb *cli.CommandBuilder[certificateConditionConfig], client cli.Destination[rootConfig, clientConfig], forceDescription string) {
	cb.Access.Value(client, &cb.Config.Client)
	cb.Access.Stdin(&cb.Config.Stdin)
	cb.Access.Stdout(&cb.Config.Stdout)

	names := cb.Values.Sequence(&cb.Config.Names, cli.Value.String())
	cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[certificateConditionConfig]) {})
	cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[certificateConditionConfig]) {
		form.Sequence(names, "NAME", cli.Count.OneOrMore())
	})

	filename := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
	filenameFlag := cb.Flags.Sequence(filename).
		DelimitedBy(",").
		Description("Filename, directory, or URL to files identifying the resource to update")
	filenameFlag.Long("filename")
	filenameFlag.Short('f')

	cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Force)).
		Description(forceDescription).
		Long("force").
		ImplicitValue(true)

	kustomize := cb.Values.String(&cb.Config.Kustomize)
	kustomizeFlag := cb.Flags.ValueFor(kustomize).
		Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
	kustomizeFlag.Long("kustomize")
	kustomizeFlag.Short('k')

	declarePrintFlags(
		cb,
		&cb.Config.Output,
		&cb.Config.OutputState,
		&cb.Config.Template,
		&cb.Config.AllowMissingTemplateKeys,
		&cb.Config.ShowManagedFields,
	)

	recursive := cb.Values.Bool(&cb.Config.Recursive)
	recursiveFlag := cb.Flags.ValueFor(recursive).
		Description("Process the directory used in -f, --filename recursively.")
	recursiveFlag.Long("recursive").ImplicitValue(true)
	recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

	cb.Rule.ExactlyOne(names, filename).And(kustomize)
	cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
}
