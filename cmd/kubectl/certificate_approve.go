package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/api/certificates/v1"
)

func certificateApproveCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[certificateConditionConfig] {
	change := certificateConditionChange{
		forbidden: v1.CertificateDenied,
		condition: v1.CertificateApproved,
		operation: "approved",
		reason:    "KubectlApprove",
		message:   "This CSR was approved by kubectl certificate approve.",
	}
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
		Run: func(ctx context.Context, cfg *certificateConditionConfig) error {
			return modifyCertificateCondition(ctx, cfg, change)
		},
	}
	return cmd
}
