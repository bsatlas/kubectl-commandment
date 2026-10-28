package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/exec"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runExec(_ context.Context, cfg *execConfig) error {
	if cfg.Resource == "" && len(cfg.Filename) == 0 {
		return fmt.Errorf("pod, type/name or --filename must be specified")
	}
	if cfg.PodRunningTimeout <= 0 {
		return fmt.Errorf("--pod-running-timeout must be higher than zero")
	}

	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	filenames := resource.FilenameOptions{
		Filenames: cfg.Filename,
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		FilenameParam(enforceNamespace, &filenames).
		NamespaceParam(namespace).DefaultNamespace()
	if cfg.Resource != "" {
		builder = builder.ResourceNames("pods", cfg.Resource)
	}
	object, err := builder.Do().Object()
	if err != nil {
		return err
	}
	if meta.IsListType(object) {
		return fmt.Errorf("cannot exec into multiple objects at a time")
	}
	pod, err := polymorphichelpers.AttachablePodForObjectFn(factory, object, cfg.PodRunningTimeout)
	if err != nil {
		return err
	}

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}

	options := exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			Namespace:     namespace,
			PodName:       pod.Name,
			ContainerName: cfg.Container,
			Stdin:         cfg.Stdin,
			TTY:           cfg.TTY,
			Quiet:         cfg.Quiet,
			IOStreams:     cfg.Streams,
		},
		Command:       append([]string(nil), cfg.Command...),
		Executor:      &exec.DefaultRemoteExecutor{},
		PodClient:     clientset.CoreV1(),
		GetPodTimeout: cfg.PodRunningTimeout,
		Config:        restConfig,
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}
