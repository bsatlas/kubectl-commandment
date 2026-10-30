package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/api/certificates/v1"
)

func certificateDenyCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[certificateConditionConfig] {
	change := certificateConditionChange{
		forbidden: v1.CertificateApproved,
		condition: v1.CertificateDenied,
		operation: "denied",
		reason:    "KubectlDeny",
		message:   "This CSR was denied by kubectl certificate deny.",
	}
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
		Run: func(ctx context.Context, cfg *certificateConditionConfig) error {
			return modifyCertificateCondition(ctx, cfg, change)
		},
	}
	return cmd
}
