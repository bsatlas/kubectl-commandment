package main

import (
	"context"
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type certificateConditionConfig struct {
	Client clientConfig
	Stdin  io.Reader
	Stdout io.Writer

	Names                    []string
	AllowMissingTemplateKeys bool
	Filename                 filenameOptions
	Force                    bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
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

	filenames := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
	filenameFlag := cb.Flags.Sequence(filenames).
		Description("Filename, directory, or URL to files identifying the resource to update").
		CompleteWith(cli.CompletionNativePaths)
	filenameFlag.Long("filename")
	filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

	force := cb.Values.Bool(&cb.Config.Force)
	forceFlag := cb.Flags.ValueFor(force)
	forceFlag.Description(forceDescription).Long("force").ImplicitValue(true)
	cb.Values.ResolveWith(forceFlag, cli.Value.KeepLast())

	kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
	kustomizeFlag := cb.Flags.ValueFor(kustomize).
		Description("Process the kustomization directory. This flag can't be used together with -f or -R.").
		CompleteWith(cli.CompletionNativeDirectories)
	kustomizeFlag.Long("kustomize")
	kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
	cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())

	declarePrintFlags(
		cb,
		&cb.Config.Output,
		&cb.Config.OutputState,
		&cb.Config.Template,
		&cb.Config.AllowMissingTemplateKeys,
		&cb.Config.ShowManagedFields,
	)

	recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
	recursiveFlag := cb.Flags.ValueFor(recursive).
		Description("Process the directory used in -f, --filename recursively.")
	recursiveFlag.Long("recursive").ImplicitValue(true)
	recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
	cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())

	cb.Rule.Check("certificate-source", func(_ context.Context, cfg *certificateConditionConfig, _ cli.Dependencies) (bool, error) {
		return len(cfg.Names) != 0 || len(cfg.Filename.Filenames()) != 0 || cfg.Filename.Kustomize != "", nil
	}).Subject(names).Subject(filenames).Subject(kustomize)
	cb.Rule.Check("certificate-filename-kustomize", func(_ context.Context, cfg *certificateConditionConfig, _ cli.Dependencies) (bool, error) {
		return len(cfg.Filename.Filenames()) == 0 || cfg.Filename.Kustomize == "", nil
	}).Subject(filenames).Subject(kustomize)
	cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
}
