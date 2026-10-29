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
			declareCertificateConditionCommand(cb, client, "Update the CSR even if it is already denied.")
		},
		Run: runCertificateDeny,
	}
	return cmd
}
