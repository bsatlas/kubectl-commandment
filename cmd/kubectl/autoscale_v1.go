package main

import (
	"context"
	"fmt"

	"k8s.io/api/autoscaling/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
)

func runAutoscaleV1(ctx context.Context, cfg *autoscaleConfig, target autoscaleTarget, mode autoscaleDryRunMode, recorder genericclioptions.Recorder, client kubernetes.Interface) error {
	name := cfg.Name
	if name == "" {
		name = target.name
	}
	hpa := &v1.HorizontalPodAutoscaler{
		Spec: v1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v1.CrossVersionObjectReference{
				APIVersion: target.mapping.GroupVersionKind.GroupVersion().String(),
				Kind:       target.mapping.GroupVersionKind.Kind,
				Name:       target.name,
			},
			MaxReplicas: cfg.Max,
		},
	}
	hpa.Name = name
	if cfg.Min > 0 {
		hpa.Spec.MinReplicas = new(cfg.Min)
	}
	if cfg.CPUPercent >= 0 {
		hpa.Spec.TargetCPUUtilizationPercentage = new(cfg.CPUPercent)
	}
	if target.enforceNamespace {
		hpa.Namespace = target.namespace
	}
	if err := recorder.Record(hpa); err != nil {
		return fmt.Errorf("error recording current command: %w", err)
	}
	if mode == autoscaleDryRunClient {
		printObject, err := autoscalePrint(cfg, "created", mode)
		if err != nil {
			return err
		}
		return printObject(hpa)
	}
	if err := util.CreateOrUpdateAnnotation(cfg.SaveConfig, hpa, scheme.DefaultJSONEncoder()); err != nil {
		return err
	}
	actual, err := client.AutoscalingV1().HorizontalPodAutoscalers(target.namespace).Create(ctx, hpa, autoscaleCreateOptions(cfg.FieldManager, mode))
	if err != nil {
		return err
	}
	printObject, err := autoscalePrint(cfg, "autoscaled", mode)
	if err != nil {
		return err
	}
	return printObject(actual)
}
