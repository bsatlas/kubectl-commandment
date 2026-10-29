package main

import (
	"context"

	"k8s.io/api/certificates/v1"
)

func runCertificateApprove(ctx context.Context, cfg *certificateConditionConfig) error {
	change := certificateConditionChange{
		forbidden: v1.CertificateDenied,
		condition: v1.CertificateApproved,
		operation: "approved",
		reason:    "KubectlApprove",
		message:   "This CSR was approved by kubectl certificate approve.",
	}
	return modifyCertificateCondition(ctx, cfg, change)
}
