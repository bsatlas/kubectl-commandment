package kubectl

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

func certificateApproveCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[certificateConditionConfig] {
	cmd := cli.Command[certificateConditionConfig]{
		Name: "approve",
		Description: cli.Description{
			Short: "Approve a certificate signing request",
			Long:  "Approve a certificate signing request.\n\n kubectl certificate approve allows a cluster admin to approve a certificate signing request (CSR). This action tells a certificate signing controller to issue a certificate to the requester with the attributes requested in the CSR.\n\n SECURITY NOTICE: Depending on the requested attributes, the issued certificate can potentially grant a requester access to cluster resources or to authenticate as a requested identity. Before approving a CSR, ensure you understand what the signed certificate can do.",
		},
		Examples: []cli.Example{
			{
				Description: "Approve CSR 'csr-sqgzp'",
				CommandLine: "kubectl certificate approve csr-sqgzp",
			},
		},
		Declare: func(cb *cli.CommandBuilder[certificateConditionConfig]) {
			declareCertificateConditionCommand(cb, client, "Update the CSR even if it is already approved.")
		},
		Run: runCertificateApprove,
	}
	return cmd
}

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
			declareCertificateConditionCommand(cb, client, "Update the CSR even if it is already denied.")
		},
		Run: runCertificateDeny,
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
