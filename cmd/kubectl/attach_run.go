package main

import (
	"context"
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/cmd/attach"
	"k8s.io/kubectl/pkg/cmd/exec"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runAttach(_ context.Context, cfg *attachConfig) error {
	if cfg.PodRunningTimeout <= 0 {
		return fmt.Errorf("--pod-running-timeout must be higher than zero")
	}

	factory := cfg.Client.factory()
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(namespace).DefaultNamespace()
	switch len(cfg.Arguments) {
	case 1:
		builder = builder.ResourceNames("pods", cfg.Arguments[0])
	case 2:
		builder = builder.ResourceNames(cfg.Arguments[0], cfg.Arguments[1])
	default:
		return fmt.Errorf("expected POD, TYPE/NAME, or TYPE NAME")
	}
	object, err := builder.Do().Object()
	if err != nil {
		return err
	}
	pod, err := polymorphichelpers.AttachablePodForObjectFn(factory, object, cfg.PodRunningTimeout)
	if err != nil {
		return err
	}
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return fmt.Errorf("cannot attach a container in a completed pod; current phase is %s", pod.Status.Phase)
	}

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}

	options := attach.AttachOptions{
		StreamOptions: exec.StreamOptions{
			Namespace:     namespace,
			ContainerName: cfg.Container,
			Stdin:         cfg.Stdin,
			TTY:           cfg.TTY,
			Quiet:         cfg.Quiet,
			IOStreams:     cfg.Streams,
		},
		DetachKeys:    cfg.DetachKeys,
		CommandName:   "kubectl attach",
		Pod:           pod,
		AttachFunc:    attach.DefaultAttachFunc,
		Resources:     append([]string(nil), cfg.Arguments...),
		Attach:        &attach.DefaultRemoteAttach{},
		GetPodTimeout: cfg.PodRunningTimeout,
		Config:        restConfig,
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}
