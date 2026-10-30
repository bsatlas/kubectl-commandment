package main

import (
	"context"
	"fmt"

	"k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/scheme"
)

const certificateConditionConflictRetries = 10

type certificateConditionChange struct {
	forbidden v1.RequestConditionType
	condition v1.RequestConditionType
	operation string
	reason    string
	message   string
}

func modifyCertificateCondition(ctx context.Context, cfg *certificateConditionConfig, change certificateConditionChange) error {
	printFlags := genericclioptions.NewPrintFlags(change.operation).WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	printer, err := printFlags.ToPrinter()
	if err != nil {
		return err
	}

	factory := cfg.Client.factoryForInvocation(ctx)
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	filenameOptions := resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}

	var found int
	builder := factory.NewBuilder().
		Unstructured().
		ContinueOnError()
	addFilenameSources(builder, false, filenameOptions, cfg.Stdin)
	result := builder.
		ResourceNames("certificatesigningrequests", cfg.Names...).
		RequireObject(true).
		Flatten().
		Latest().
		Do()
	err = result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		for attempt := range certificateConditionConflictRetries + 1 {
			object, ok := info.Object.(*unstructured.Unstructured)
			if !ok {
				return fmt.Errorf("expected *unstructured.Unstructured, got %T", info.Object)
			}
			if want, got := v1.Kind("CertificateSigningRequest"), object.GetObjectKind().GroupVersionKind().GroupKind(); want != got {
				return fmt.Errorf("can only handle %s objects, got %s", want.String(), got.String())
			}
			request, err := clientset.CertificatesV1().CertificateSigningRequests().Get(ctx, object.GetName(), getOptions())
			if errors.IsNotFound(err) {
				return fmt.Errorf("could not find v1 version of %s: %w", object.GetName(), err)
			}
			if err != nil {
				return err
			}
			already, err := addCertificateCondition(request, change)
			if err != nil {
				return err
			}
			if !already || cfg.Force {
				_, err = clientset.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, request.Name, request, updateOptions())
				if errors.IsConflict(err) && attempt < certificateConditionConflictRetries {
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
		return printer.PrintObj(info.Object, cfg.Stdout)
	})
	if found == 0 && err == nil {
		_, err = fmt.Fprintln(cfg.Stdout, "No resources found")
	}
	return err
}

func addCertificateCondition(request *v1.CertificateSigningRequest, change certificateConditionChange) (bool, error) {
	var already bool
	for _, existing := range request.Status.Conditions {
		if existing.Type == change.forbidden {
			return false, fmt.Errorf("certificate signing request %q is already %s", request.Name, existing.Type)
		}
		if existing.Type == change.condition {
			already = true
		}
	}
	if already {
		return true, nil
	}
	condition := v1.CertificateSigningRequestCondition{
		Type:           change.condition,
		Status:         "True",
		Reason:         change.reason,
		Message:        change.message,
		LastUpdateTime: metaNow(),
	}
	request.Status.Conditions = append(request.Status.Conditions, condition)
	return false, nil
}
