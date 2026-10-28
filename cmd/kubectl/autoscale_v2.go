package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/api/autoscaling/v2"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
)

func runAutoscaleV2(ctx context.Context, cfg *autoscaleConfig, target autoscaleTarget, mode autoscaleDryRunMode, recorder genericclioptions.Recorder, client kubernetes.Interface) error {
	hpa, err := autoscalerV2(cfg, target)
	if err != nil {
		return err
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
	actual, err := client.AutoscalingV2().HorizontalPodAutoscalers(target.namespace).Create(ctx, hpa, autoscaleCreateOptions(cfg.FieldManager, mode))
	if err != nil {
		return err
	}
	printObject, err := autoscalePrint(cfg, "autoscaled", mode)
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
		metrics = append(metrics, v2.MetricSpec{
			Type: v2.ResourceMetricSourceType,
			Resource: &v2.ResourceMetricSource{
				Name: "cpu",
				Target: v2.MetricTarget{
					Type:               v2.UtilizationMetricType,
					AverageUtilization: new(cfg.CPUPercent),
				},
			},
		})
	}
	if cfg.CPU != "" {
		metric, err := autoscaleMetric(cfg.CPU, "cpu")
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	if cfg.Memory != "" {
		metric, err := autoscaleMetric(cfg.Memory, "memory")
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	hpa.Spec.Metrics = metrics
	return hpa, nil
}

func autoscaleMetric(input, resourceName string) (v2.MetricSpec, error) {
	quantity, value, metricType, err := parseAutoscaleResource(input, resourceName)
	if err != nil {
		return v2.MetricSpec{}, err
	}
	target := v2.MetricTarget{
		Type: metricType,
	}
	switch metricType {
	case v2.UtilizationMetricType:
		target.AverageUtilization = new(value)
	case v2.AverageValueMetricType:
		target.AverageValue = new(quantity)
	default:
		return v2.MetricSpec{}, fmt.Errorf("unsupported metric type: %v", metricType)
	}
	metric := v2.MetricSpec{
		Type: v2.ResourceMetricSourceType,
		Resource: &v2.ResourceMetricSource{
			Name:   v1.ResourceName(resourceName),
			Target: target,
		},
	}
	return metric, nil
}

func autoscaleV1Compatible(cfg *autoscaleConfig) (bool, error) {
	_, _, metricType, err := parseAutoscaleResource(cfg.CPU, "cpu")
	if err != nil {
		return false, err
	}
	return (cfg.CPUPercent >= 0 && cfg.Memory == "") || (cfg.Memory == "" && metricType == v2.UtilizationMetricType), nil
}

func parseAutoscaleResource(input, resourceName string) (resource.Quantity, int32, v2.MetricTargetType, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return resource.Quantity{}, 0, "", fmt.Errorf("empty input")
	}
	if strings.HasSuffix(input, "%") {
		trimmed := strings.TrimSuffix(input, "%")
		value, err := strconv.ParseInt(trimmed, 10, 32)
		if err != nil || value < 0 {
			return resource.Quantity{}, 0, "", fmt.Errorf("invalid percentage value: %s", trimmed)
		}
		return resource.Quantity{}, int32(value), v2.UtilizationMetricType, nil
	}
	if value, err := strconv.ParseFloat(input, 64); err == nil {
		unit := ""
		switch resourceName {
		case "cpu":
			unit = "m"
		case "memory":
			unit = "Mi"
		default:
			return resource.Quantity{}, 0, "", fmt.Errorf("unsupported resource type: %v", resourceName)
		}
		quantity, err := resource.ParseQuantity(fmt.Sprintf("%g%s", value, unit))
		if err != nil {
			return resource.Quantity{}, 0, "", err
		}
		return quantity, 0, v2.AverageValueMetricType, nil
	}
	quantity, err := resource.ParseQuantity(input)
	if err != nil {
		return resource.Quantity{}, 0, "", fmt.Errorf("invalid resource %s value: %s", resourceName, input)
	}
	return quantity, 0, v2.AverageValueMetricType, nil
}
