package main

import (
	"context"
	"fmt"

	"k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/scheme"
)

type certificateConditionChange struct {
	forbidden v1.RequestConditionType
	condition v1.RequestConditionType
	reason    string
	message   string
}

func runCertificateApprove(ctx context.Context, cfg *certificateConditionConfig) error {
	change := certificateConditionChange{
		forbidden: v1.CertificateDenied,
		condition: v1.CertificateApproved,
		reason:    "KubectlApprove",
		message:   "This CSR was approved by kubectl certificate approve.",
	}
	return modifyCertificateCondition(ctx, cfg, change)
}

func runCertificateDeny(ctx context.Context, cfg *certificateConditionConfig) error {
	change := certificateConditionChange{
		forbidden: v1.CertificateApproved,
		condition: v1.CertificateDenied,
		reason:    "KubectlDeny",
		message:   "This CSR was denied by kubectl certificate deny.",
	}
	return modifyCertificateCondition(ctx, cfg, change)
}

func modifyCertificateCondition(ctx context.Context, cfg *certificateConditionConfig, change certificateConditionChange) error {
	if len(cfg.Arguments) == 0 && len(cfg.Filename) == 0 && cfg.Kustomize == "" {
		return fmt.Errorf("one or more CSRs must be specified as <name> or -f <filename>")
	}

	getter := restClientGetter(cfg.Client)
	clientset, err := cfg.Client.factory().KubernetesClientSet()
	if err != nil {
		return err
	}
	filenameOptions := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	printFlags := genericclioptions.NewPrintFlags(conditionLabel(change.condition)).WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printer, err := printFlags.ToPrinter()
	if err != nil {
		return err
	}

	found := 0
	result := resource.NewBuilder(getter).
		Unstructured().
		ContinueOnError().
		FilenameParam(false, &filenameOptions).
		ResourceNames("certificatesigningrequests", cfg.Arguments...).
		RequireObject(true).
		Flatten().
		Latest().
		Do()
	err = result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		for attempt := 0; ; attempt++ {
			object, ok := info.Object.(*unstructured.Unstructured)
			if !ok {
				return fmt.Errorf("expected *unstructured.Unstructured, got %T", info.Object)
			}
			if want, got := v1.Kind("CertificateSigningRequest"), object.GetObjectKind().GroupVersionKind().GroupKind(); want != got {
				return fmt.Errorf("can only handle %s objects, got %s", want.String(), got.String())
			}
			request, err := getCertificateSigningRequest(ctx, clientset, object.GetName())
			if errors.IsNotFound(err) {
				return fmt.Errorf("could not find v1 version of %s: %v", object.GetName(), err)
			}
			if err != nil {
				return err
			}
			changed, already, err := addCertificateCondition(request, change)
			if err != nil {
				return err
			}
			if !already || cfg.Force {
				err = updateCertificateSigningRequestApproval(ctx, clientset, changed)
				if errors.IsConflict(err) && attempt < 10 {
					if err := info.Get(); err != nil {
						return err
					}
					continue
				}
				if err != nil {
					return err
				}
			}
			break
		}
		found++
		return printer.PrintObj(info.Object, cfg.Streams.Out)
	})
	if found == 0 && err == nil {
		_, _ = fmt.Fprintln(cfg.Streams.Out, "No resources found")
	}
	return err
}

func addCertificateCondition(object runtime.Object, change certificateConditionChange) (*v1.CertificateSigningRequest, bool, error) {
	request, ok := object.(*v1.CertificateSigningRequest)
	if !ok {
		return nil, false, fmt.Errorf("can only handle certificates.k8s.io CertificateSigningRequest objects, got %T", object)
	}
	already := false
	for _, existing := range request.Status.Conditions {
		if existing.Type == change.forbidden {
			return nil, false, fmt.Errorf("certificate signing request %q is already %s", request.Name, existing.Type)
		}
		if existing.Type == change.condition {
			already = true
		}
	}
	if already {
		return request, true, nil
	}
	request.Status.Conditions = append(request.Status.Conditions, v1.CertificateSigningRequestCondition{
		Type:           change.condition,
		Status:         "True",
		Reason:         change.reason,
		Message:        change.message,
		LastUpdateTime: metaNow(),
	})
	return request, false, nil
}

func conditionLabel(condition v1.RequestConditionType) string {
	if condition == v1.CertificateApproved {
		return "approved"
	}
	return "denied"
}
