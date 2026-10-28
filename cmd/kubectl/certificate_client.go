package main

import (
	"context"

	"k8s.io/api/certificates/v1"
	"k8s.io/client-go/kubernetes"
)

func getCertificateSigningRequest(ctx context.Context, client kubernetes.Interface, name string) (*v1.CertificateSigningRequest, error) {
	return client.CertificatesV1().CertificateSigningRequests().Get(ctx, name, getOptions())
}

func updateCertificateSigningRequestApproval(ctx context.Context, client kubernetes.Interface, request *v1.CertificateSigningRequest) error {
	_, err := client.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, request.Name, request, updateOptions())
	return err
}
