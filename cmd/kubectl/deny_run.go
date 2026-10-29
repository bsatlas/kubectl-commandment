package main

import (
	"context"

	"k8s.io/api/certificates/v1"
)

func runCertificateDeny(ctx context.Context, cfg *certificateConditionConfig) error {
	change := certificateConditionChange{
		forbidden: v1.CertificateApproved,
		condition: v1.CertificateDenied,
		operation: "denied",
		reason:    "KubectlDeny",
		message:   "This CSR was denied by kubectl certificate deny.",
	}
	return modifyCertificateCondition(ctx, cfg, change)
}
