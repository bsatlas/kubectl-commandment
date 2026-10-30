package main

import (
	"context"
	"fmt"

	"k8s.io/api/autoscaling/v1"
	"k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/scale"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	kubectlutil "k8s.io/kubectl/pkg/util"
)

type autoscaleTarget struct {
	name             string
	mapping          *meta.RESTMapping
	namespace        string
	enforceNamespace bool
}

func autoscaleCreateOptions(fieldManager string, dryRun util.DryRunStrategy) metav1.CreateOptions {
	options := metav1.CreateOptions{
		FieldManager: fieldManager,
	}
	if dryRun == util.DryRunServer {
		options.DryRun = []string{metav1.DryRunAll}
	}
	return options
}

func runAutoscale(ctx context.Context, cfg *autoscaleConfig) error {
	factory := cfg.Client.factory()
	dryRun := cfg.DryRun
	discovery, err := factory.ToDiscoveryClient()
	if err != nil {
		return err
	}
	resolver := scale.NewDiscoveryScaleKindResolver(discovery)
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	recorder, err := autoscaleRecorder(cfg.Record)
	if err != nil {
		return err
	}
	filename := &resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	result := factory.NewBuilder().
		Unstructured().
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, filename).
		ResourceTypeOrNameArgs(false, cfg.Arguments...).
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	count := 0
	err = result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		mapping := info.ResourceMapping()
		gvr := mapping.GroupVersionKind.GroupVersion().WithResource(mapping.Resource.Resource)
		if _, err := resolver.ScaleForResource(gvr); err != nil {
			return fmt.Errorf("cannot autoscale a %s: %w", mapping.GroupVersionKind.Kind, err)
		}

		target := autoscaleTarget{
			name:             info.Name,
			mapping:          mapping,
			namespace:        namespace,
			enforceNamespace: enforceNamespace,
		}
		if err := runAutoscaleV2(ctx, cfg, target, dryRun, recorder, clientset); err != nil {
			klog.V(1).Infof("Encountered an error with the autoscaling/v2 HorizontalPodAutoscaler: %v. Falling back to try the autoscaling/v1 HorizontalPodAutoscaler", err)
			compatible, compatibilityErr := autoscaleV1Compatible(cfg)
			if !compatible {
				return fmt.Errorf("failed to create autoscaling/v2 HPA and the configuration is incompatible with autoscaling/v1: %w", compatibilityErr)
			}
			if err := runAutoscaleV1(ctx, cfg, target, dryRun, recorder, clientset); err != nil {
				return err
			}
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no objects passed to autoscale")
	}
	return nil
}

func autoscaleRecorder(record bool) (genericclioptions.Recorder, error) {
	flags := genericclioptions.NewRecordFlags()
	*flags.Record = record
	if record {
		if err := flags.CompleteWithChangeCause("kubectl autoscale"); err != nil {
			return nil, err
		}
	}
	return flags.ToRecorder()
}

func autoscalePrint(cfg *autoscaleConfig, operation string, strategy util.DryRunStrategy) (func(runtime.Object) error, error) {
	flags := genericclioptions.NewPrintFlags(operation).WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(flags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	util.PrintFlagsWithDryRunStrategy(flags, strategy)
	printer, err := flags.ToPrinter()
	if err != nil {
		return nil, err
	}
	return func(object runtime.Object) error {
		return printer.PrintObj(object, cfg.Streams.Out)
	}, nil
}

func runAutoscaleV1(ctx context.Context, cfg *autoscaleConfig, target autoscaleTarget, dryRun util.DryRunStrategy, recorder genericclioptions.Recorder, client kubernetes.Interface) error {
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
	if dryRun == util.DryRunClient {
		printObject, err := autoscalePrint(cfg, "created", dryRun)
		if err != nil {
			return err
		}
		return printObject(hpa)
	}
	if err := kubectlutil.CreateOrUpdateAnnotation(cfg.SaveConfig, hpa, scheme.DefaultJSONEncoder()); err != nil {
		return err
	}
	actual, err := client.AutoscalingV1().HorizontalPodAutoscalers(target.namespace).Create(ctx, hpa, autoscaleCreateOptions(cfg.FieldManager, dryRun))
	if err != nil {
		return err
	}
	printObject, err := autoscalePrint(cfg, "autoscaled", dryRun)
	if err != nil {
		return err
	}
	return printObject(actual)
}

func runAutoscaleV2(ctx context.Context, cfg *autoscaleConfig, target autoscaleTarget, dryRun util.DryRunStrategy, recorder genericclioptions.Recorder, client kubernetes.Interface) error {
	hpa, err := autoscalerV2(cfg, target)
	if err != nil {
		return err
	}
	if err := recorder.Record(hpa); err != nil {
		return fmt.Errorf("error recording current command: %w", err)
	}
	if dryRun == util.DryRunClient {
		printObject, err := autoscalePrint(cfg, "created", dryRun)
		if err != nil {
			return err
		}
		return printObject(hpa)
	}
	if err := kubectlutil.CreateOrUpdateAnnotation(cfg.SaveConfig, hpa, scheme.DefaultJSONEncoder()); err != nil {
		return err
	}
	actual, err := client.AutoscalingV2().HorizontalPodAutoscalers(target.namespace).Create(ctx, hpa, autoscaleCreateOptions(cfg.FieldManager, dryRun))
	if err != nil {
		return err
	}
	printObject, err := autoscalePrint(cfg, "autoscaled", dryRun)
	if err != nil {
		return err
	}
	return printObject(actual)
}

func autoscalerV2(cfg *autoscaleConfig, target autoscaleTarget) (*v2.HorizontalPodAutoscaler, error) {
	name := cfg.Name
	if name == "" {
		name = target.name
	}
	hpa := &v2.HorizontalPodAutoscaler{
		Spec: v2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2.CrossVersionObjectReference{
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
	if target.enforceNamespace {
		hpa.Namespace = target.namespace
	}

	var metrics []v2.MetricSpec
	if cfg.CPUPercent > 0 {
		target := v2.MetricTarget{
			Type:               v2.UtilizationMetricType,
			AverageUtilization: new(cfg.CPUPercent),
		}
		resourceMetric := &v2.ResourceMetricSource{
			Name:   "cpu",
			Target: target,
		}
		metric := v2.MetricSpec{
			Type:     v2.ResourceMetricSourceType,
			Resource: resourceMetric,
		}
		metrics = append(metrics, metric)
	}
	if cfg.CPU.Type != "" {
		metric, err := autoscaleMetric(cfg.CPU, "cpu")
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	if cfg.Memory.Type != "" {
		metric, err := autoscaleMetric(cfg.Memory, "memory")
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	hpa.Spec.Metrics = metrics
	return hpa, nil
}

func autoscaleMetric(input autoscaleMetricTarget, resourceName string) (v2.MetricSpec, error) {
	target := v2.MetricTarget{
		Type: input.Type,
	}
	switch input.Type {
	case v2.UtilizationMetricType:
		target.AverageUtilization = new(input.Utilization)
	case v2.AverageValueMetricType:
		target.AverageValue = new(input.Quantity.DeepCopy())
	default:
		return v2.MetricSpec{}, fmt.Errorf("unsupported metric type: %v", input.Type)
	}
	resourceMetric := &v2.ResourceMetricSource{
		Target: target,
	}
	switch resourceName {
	case "cpu":
		resourceMetric.Name = "cpu"
	case "memory":
		resourceMetric.Name = "memory"
	default:
		return v2.MetricSpec{}, fmt.Errorf("unsupported resource type: %v", resourceName)
	}
	metric := v2.MetricSpec{
		Type:     v2.ResourceMetricSourceType,
		Resource: resourceMetric,
	}
	return metric, nil
}

func autoscaleV1Compatible(cfg *autoscaleConfig) (bool, error) {
	if cfg.CPU.Type == "" {
		return false, fmt.Errorf("empty input")
	}
	return (cfg.CPUPercent >= 0 && cfg.Memory.Type == "") || (cfg.Memory.Type == "" && cfg.CPU.Type == v2.UtilizationMetricType), nil
}
