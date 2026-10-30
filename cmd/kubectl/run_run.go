/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiwatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/attach"
	"k8s.io/kubectl/pkg/cmd/delete"
	"k8s.io/kubectl/pkg/cmd/exec"
	"k8s.io/kubectl/pkg/cmd/logs"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/interrupt"
	utilsexec "k8s.io/utils/exec"
)

var errRunPodCompleted = errors.New("pod ran to completion")

func runPodCompleted(event apiwatch.Event) (bool, error) {
	if event.Type == apiwatch.Deleted {
		return false, runDeletedPodError()
	}
	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		return false, nil
	}
	return pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded, nil
}

func runPodSucceeded(event apiwatch.Event) (bool, error) {
	if event.Type == apiwatch.Deleted {
		return false, runDeletedPodError()
	}
	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		return false, nil
	}
	return pod.Status.Phase == v1.PodSucceeded, nil
}

func runPodRunningAndReady(event apiwatch.Event) (bool, error) {
	if event.Type == apiwatch.Deleted {
		return false, runDeletedPodError()
	}
	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		return false, nil
	}
	switch pod.Status.Phase {
	case v1.PodFailed, v1.PodSucceeded:
		return false, errRunPodCompleted
	case v1.PodRunning:
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				return true, nil
			}
		}
	}
	return false, nil
}

type runObject struct {
	Object  runtime.Object
	Mapping *meta.RESTMapping
}

type runObjectCreator struct {
	factory          util.Factory
	config           *runConfig
	recorder         genericclioptions.Recorder
	strategy         util.DryRunStrategy
	namespace        string
	enforceNamespace bool
}

func runRun(ctx context.Context, cfg *runConfig) error {
	explicitLabels := cfg.Labels
	labels := explicitLabels
	if len(labels) == 0 {
		labels = map[string]string{
			"run": cfg.Name,
		}
	}
	annotations := make(map[string]string, len(cfg.Annotations))
	for _, annotation := range cfg.Annotations {
		annotations[annotation.Name] = annotation.Value
	}
	environment := slices.Clone(cfg.Env)

	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	printObject, err := printObject(printFlags, cfg.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl run"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		return err
	}
	deleteFlags := delete.NewDeleteFlags("to use to replace the resource.")
	*deleteFlags.CascadingStrategy = "background"
	*deleteFlags.Force = cfg.Force
	*deleteFlags.GracePeriod = cfg.GracePeriod
	*deleteFlags.Timeout = cfg.Timeout
	*deleteFlags.Wait = cfg.Wait
	*deleteFlags.FileNameFlags.Filenames = slices.Concat(cfg.Filename...)
	*deleteFlags.FileNameFlags.Kustomize = cfg.Kustomize
	*deleteFlags.FileNameFlags.Recursive = cfg.Recursive
	deleteOptions, err := deleteFlags.ToOptions(dynamicClient, cfg.Streams)
	if err != nil {
		return err
	}
	deleteOptions.CascadingStrategy = metav1.DeletionPropagation(cfg.Cascade.Strategy)
	if cfg.Cascade.Deprecated {
		fmt.Fprintf(cfg.Streams.ErrOut, "warning: --cascade=%v is deprecated (boolean value) and can be replaced with --cascade=%s.\n", cfg.Cascade.Source, cfg.Cascade.Strategy)
	}
	deleteOptions.IgnoreNotFound = true
	deleteOptions.WaitForDeletion = false
	deleteOptions.GracePeriod = -1
	deleteOptions.Quiet = cfg.Quiet

	creator := runObjectCreator{
		factory:          factory,
		config:           cfg,
		recorder:         recorder,
		strategy:         cfg.DryRunStrategy,
		namespace:        namespace,
		enforceNamespace: enforceNamespace,
	}
	pod := createRunPod(cfg, labels, annotations, environment)
	createdPod, err := creator.create(pod, &v1.Pod{})
	if err != nil {
		return err
	}
	created := []*runObject{createdPod}

	var failures []error
	if cfg.Expose {
		service := createRunService(cfg, labels, explicitLabels)
		createdService, createErr := creator.create(service, nil)
		if createErr != nil {
			failures = append(failures, createErr)
		} else {
			created = append(created, createdService)
		}
	}

	if cfg.Attach {
		if cfg.Rm {
			defer func() {
				_ = removeRunObjects(factory, deleteOptions, created)
			}()
		}
		return runAttached(ctx, cfg, factory, createdPod, cfg.RestartPolicy)
	}

	for _, object := range created {
		if err := printObject(object.Object); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (creator runObjectCreator) create(object, overrideData runtime.Object) (*runObject, error) {
	cfg := creator.config
	mapper, err := creator.factory.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	kinds, _, err := scheme.Scheme.ObjectKinds(object)
	if err != nil {
		return nil, err
	}
	mapping, err := mapper.RESTMapping(kinds[0].GroupKind(), kinds[0].Version)
	if err != nil {
		return nil, err
	}

	if overrideData != nil && cfg.Overrides != "" {
		object, err = applyOverrides(object, overrideData, cfg.Overrides, string(cfg.OverrideType))
		if err != nil {
			return nil, err
		}
	}
	if err := creator.recorder.Record(object); err != nil {
		klog.V(4).Infof("error recording current command: %v", err)
	}

	actual := object
	if creator.strategy != util.DryRunClient {
		if err := saveConfigAnnotation(cfg.SaveConfig, object); err != nil {
			return nil, err
		}
		client, err := creator.factory.ClientForMapping(mapping)
		if err != nil {
			return nil, err
		}
		actual, err = resource.NewHelper(client, mapping).
			DryRun(creator.strategy == util.DryRunServer).
			WithFieldManager(cfg.FieldManager).
			Create(creator.namespace, false, object)
		if err != nil {
			return nil, err
		}
	} else if accessor, accessorErr := meta.Accessor(actual); accessorErr == nil && creator.enforceNamespace {
		accessor.SetNamespace(creator.namespace)
	}

	result := runObject{
		Object:  actual,
		Mapping: mapping,
	}
	return &result, nil
}

func createRunPod(cfg *runConfig, labels, annotations map[string]string, environment []v1.EnvVar) *v1.Pod {
	var securityContext *v1.SecurityContext
	if cfg.Privileged {
		securityContext = &v1.SecurityContext{
			Privileged: &cfg.Privileged,
		}
	}
	container := v1.Container{
		Name:            cfg.Name,
		Image:           cfg.Image,
		Env:             environment,
		Stdin:           cfg.Stdin,
		StdinOnce:       !cfg.LeaveStdinOpen && cfg.Stdin,
		TTY:             cfg.TTY,
		ImagePullPolicy: cfg.ImagePullPolicy,
		SecurityContext: securityContext,
	}
	if len(cfg.Arguments) > 0 {
		if cfg.Command {
			container.Command = slices.Clone(cfg.Arguments)
		} else {
			container.Args = slices.Clone(cfg.Arguments)
		}
	}
	if cfg.Port != nil {
		container.Ports = []v1.ContainerPort{{
			ContainerPort: int32(*cfg.Port),
		}}
	}
	pod := v1.Pod{
		Spec: v1.PodSpec{
			Containers:    []v1.Container{container},
			DNSPolicy:     v1.DNSClusterFirst,
			RestartPolicy: cfg.RestartPolicy,
		},
	}
	pod.Name = cfg.Name
	pod.Labels = labels
	pod.Annotations = annotations
	return &pod
}

func createRunService(cfg *runConfig, selector, labels map[string]string) *v1.Service {
	port := *cfg.Port
	service := v1.Service{
		Spec: v1.ServiceSpec{
			Selector: selector,
			Ports: []v1.ServicePort{{
				Port:       int32(port),
				Protocol:   v1.ProtocolTCP,
				TargetPort: intstr.FromInt(port),
			}},
		},
	}
	service.Name = cfg.Name
	service.Labels = labels
	return &service
}

func runAttached(ctx context.Context, cfg *runConfig, factory util.Factory, created *runObject, restartPolicy v1.RestartPolicy) error {
	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	pod, err := polymorphichelpers.AttachablePodForObjectFn(factory, created.Object, cfg.PodRunningTimeout)
	if err != nil {
		return err
	}
	options := attach.AttachOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: cfg.Streams,
			Stdin:     cfg.Stdin,
			TTY:       cfg.TTY,
			Quiet:     cfg.Quiet,
		},
		CommandName:   "kubectl attach",
		GetPodTimeout: cfg.PodRunningTimeout,
		Attach:        &attach.DefaultRemoteAttach{},
		DetachKeys:    cfg.DetachKeys,
		Config:        restConfig,
		AttachFunc:    attach.DefaultAttachFunc,
	}
	if err := handleRunAttach(ctx, factory, clientset, pod.Namespace, pod.Name, &options); err != nil {
		return err
	}
	if cfg.LeaveStdinOpen || (restartPolicy != v1.RestartPolicyNever && restartPolicy != v1.RestartPolicyOnFailure) {
		return nil
	}

	condition := runPodCompleted
	if restartPolicy == v1.RestartPolicyOnFailure {
		condition = runPodSucceeded
	}
	finished, err := waitForRunPod(ctx, clientset, pod.Namespace, pod.Name, cfg.PodRunningTimeout, condition)
	if err != nil {
		return err
	}
	switch finished.Status.Phase {
	case v1.PodSucceeded:
		return nil
	case v1.PodFailed:
		if len(finished.Status.ContainerStatuses) == 0 || finished.Status.ContainerStatuses[0].State.Terminated == nil {
			return fmt.Errorf("pod %s/%s failed with unknown exit code", finished.Namespace, finished.Name)
		}
		terminated := finished.Status.ContainerStatuses[0].State.Terminated
		if terminated.ExitCode == 0 {
			return fmt.Errorf("pod %s/%s failed with unknown exit code", finished.Namespace, finished.Name)
		}
		message := fmt.Errorf("pod %s/%s terminated (%s)\n%s", finished.Namespace, finished.Name, terminated.Reason, terminated.Message)
		failure := utilsexec.CodeExitError{
			Err:  message,
			Code: int(terminated.ExitCode),
		}
		return failure
	default:
		return fmt.Errorf("pod %s/%s left in phase %s", finished.Namespace, finished.Name, finished.Status.Phase)
	}
}

func removeRunObjects(factory util.Factory, options *delete.DeleteOptions, objects []*runObject) error {
	accessor := meta.NewAccessor()
	for _, object := range objects {
		namespace, err := accessor.Namespace(object.Object)
		if err != nil {
			return err
		}
		name, err := accessor.Name(object.Object)
		if err != nil {
			return err
		}
		result := factory.NewBuilder().
			WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
			ContinueOnError().
			NamespaceParam(namespace).DefaultNamespace().
			ResourceNames(object.Mapping.Resource.Resource+"."+object.Mapping.Resource.Group, name).
			Flatten().
			Do()
		if err := options.DeleteResult(result); err != nil {
			return err
		}
	}
	return nil
}

func handleRunAttach(ctx context.Context, factory util.Factory, clientset kubernetes.Interface, namespace, name string, options *attach.AttachOptions) error {
	pod, err := waitForRunPod(ctx, clientset, namespace, name, options.GetPodTimeout, runPodRunningAndReady)
	if err != nil && !errors.Is(err, errRunPodCompleted) {
		return err
	}
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return runLogOptions(ctx, factory, pod, options, nil)
	}
	options.Pod = pod
	options.PodName = name
	options.Namespace = namespace

	containerName, containerErr := options.GetContainerName(pod)
	var logsSince *v1.PodLogOptions
	if containerErr == nil {
		logContext, cancel := context.WithTimeout(ctx, 5*time.Second)
		initial := &v1.PodLogOptions{
			Container: containerName,
		}
		if logErr := runLogOptions(logContext, factory, pod, options, initial); logErr == nil {
			now := metaNow()
			initial.SinceTime = &now
			logsSince = initial
		}
		cancel()
	}
	if err := options.Run(); err != nil {
		_, _ = fmt.Fprintf(options.ErrOut, "warning: couldn't attach to pod/%s, falling back to streaming logs: %v\n", name, err)
		if logsSince == nil {
			logsSince = &v1.PodLogOptions{}
		}
		logsSince.Follow = true
		return runLogOptions(ctx, factory, pod, options, logsSince)
	}
	return nil
}

func runLogOptions(ctx context.Context, factory util.Factory, pod *v1.Pod, options *attach.AttachOptions, podOptions *v1.PodLogOptions) error {
	if podOptions == nil {
		containerName, err := options.GetContainerName(pod)
		if err != nil {
			return err
		}
		podOptions = &v1.PodLogOptions{
			Container: containerName,
		}
	}
	requests, err := polymorphichelpers.LogsForObjectFn(factory, pod, podOptions, options.GetPodTimeout, false)
	if err != nil {
		return err
	}
	for _, request := range requests {
		if err := logs.DefaultConsumeRequest(ctx, request, options.Out); err != nil {
			if podOptions.Follow && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				return nil
			}
			return err
		}
	}
	return nil
}

func runDeletedPodError() error {
	resource := schema.GroupResource{
		Resource: "pods",
	}
	return apierrors.NewNotFound(resource, "")
}

func waitForRunPod(ctx context.Context, clientset kubernetes.Interface, namespace, name string, timeout time.Duration, condition watchtools.ConditionFunc) (*v1.Pod, error) {
	ctx, cancel := watchtools.ContextWithOptionalTimeout(ctx, timeout)
	defer cancel()

	selector := fields.OneTermEqualSelector("metadata.name", name).String()
	listWatch := cache.ListWatch{
		ListWithContextFunc: func(listCtx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = selector
			return clientset.CoreV1().Pods(namespace).List(listCtx, options)
		},
		WatchFuncWithContext: func(watchCtx context.Context, options metav1.ListOptions) (apiwatch.Interface, error) {
			options.FieldSelector = selector
			return clientset.CoreV1().Pods(namespace).Watch(watchCtx, options)
		},
	}

	intr := interrupt.New(nil, cancel)
	var result *v1.Pod
	err := intr.Run(func() error {
		event, err := watchtools.UntilWithSync(ctx, &listWatch, &v1.Pod{}, nil, condition)
		if event != nil {
			result, _ = event.Object.(*v1.Pod)
		}
		return err
	})
	return result, err
}
