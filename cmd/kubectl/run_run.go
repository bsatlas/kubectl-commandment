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
	"strconv"
	"strings"
	"time"

	"github.com/distribution/reference"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/attach"
	"k8s.io/kubectl/pkg/cmd/delete"
	"k8s.io/kubectl/pkg/cmd/exec"
	"k8s.io/kubectl/pkg/cmd/logs"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

var errRunPodCompleted = errors.New("pod ran to completion")

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
	attachRequested := cfg.Attach
	if !explicitFlagValue(cfg.AttachState) && cfg.Stdin {
		attachRequested = true
	}
	if cfg.Name == "" {
		return errors.New("NAME is required for run")
	}
	if cfg.Image == "" {
		return errors.New("--image is required")
	}
	if !reference.ReferenceRegexp.MatchString(cfg.Image) {
		return fmt.Errorf("invalid image name %q: %w", cfg.Image, reference.ErrReferenceInvalidFormat)
	}
	if cfg.TTY && !cfg.Stdin {
		return errors.New("-i/--stdin is required for containers with -t/--tty=true")
	}
	if cfg.Expose && cfg.Port == "" {
		return errors.New("--port must be set when exposing a service")
	}
	if cfg.Port != "" {
		if _, err := strconv.Atoi(cfg.Port); err != nil {
			return errors.New("--port must be a number")
		}
	}
	if !attachRequested && cfg.Rm {
		return errors.New("--rm should only be used for attached containers")
	}

	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	if attachRequested && strategy != util.DryRunNone {
		return errors.New("--dry-run=[server|client] can't be used with attached containers options (--attach, --stdin, or --tty)")
	}

	explicitLabels, err := parseRunLabels(cfg.Labels)
	if err != nil {
		return err
	}
	labels := explicitLabels
	if len(labels) == 0 {
		labels = map[string]string{
			"run": cfg.Name,
		}
	}
	annotations, _, err := util.ParsePairs(cfg.Annotations, "annotations", false)
	if err != nil {
		return err
	}
	environment, err := parseRunEnvironment(cfg.Env)
	if err != nil {
		return err
	}
	restartPolicy, err := runRestartPolicy(cfg.Restart, cfg.Stdin)
	if err != nil {
		return err
	}
	pullPolicy, err := runImagePullPolicy(cfg.ImagePullPolicy)
	if err != nil {
		return err
	}

	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printObject, err := printObject(printFlags, strategy, cfg.Streams.Out)
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
	deleteOptions, err := deleteFlags.ToOptions(dynamicClient, cfg.Streams)
	if err != nil {
		return err
	}
	deleteOptions.IgnoreNotFound = true
	deleteOptions.WaitForDeletion = false
	deleteOptions.GracePeriod = -1
	deleteOptions.Quiet = cfg.Quiet

	creator := runObjectCreator{
		factory:          factory,
		config:           cfg,
		recorder:         recorder,
		strategy:         strategy,
		namespace:        namespace,
		enforceNamespace: enforceNamespace,
	}
	pod := createRunPod(cfg, labels, annotations, environment, restartPolicy, pullPolicy)
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

	if attachRequested {
		if cfg.Rm {
			defer func() {
				_ = removeRunObjects(factory, deleteOptions, created)
			}()
		}
		return runAttached(ctx, cfg, factory, createdPod, restartPolicy)
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
		object, err = applyOverrides(object, overrideData, cfg.Overrides, cfg.OverrideType)
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

func createRunPod(cfg *runConfig, labels, annotations map[string]string, environment []v1.EnvVar, restartPolicy v1.RestartPolicy, pullPolicy v1.PullPolicy) *v1.Pod {
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
		ImagePullPolicy: pullPolicy,
		SecurityContext: securityContext,
	}
	if len(cfg.Arguments) > 0 {
		if cfg.Command {
			container.Command = append([]string(nil), cfg.Arguments...)
		} else {
			container.Args = append([]string(nil), cfg.Arguments...)
		}
	}
	if cfg.Port != "" {
		port, _ := strconv.Atoi(cfg.Port)
		container.Ports = []v1.ContainerPort{{
			ContainerPort: int32(port),
		}}
	}
	pod := v1.Pod{
		ObjectMeta: runObjectMeta(cfg.Name, labels, annotations),
		Spec: v1.PodSpec{
			Containers:    []v1.Container{container},
			DNSPolicy:     v1.DNSClusterFirst,
			RestartPolicy: restartPolicy,
		},
	}
	return &pod
}

func createRunService(cfg *runConfig, selector, labels map[string]string) *v1.Service {
	port, _ := strconv.Atoi(cfg.Port)
	service := v1.Service{
		ObjectMeta: runObjectMeta(cfg.Name, labels, nil),
		Spec: v1.ServiceSpec{
			Selector: selector,
			Ports: []v1.ServicePort{{
				Port:       int32(port),
				Protocol:   v1.ProtocolTCP,
				TargetPort: intstr.FromInt(port),
			}},
		},
	}
	return &service
}

func parseRunLabels(value string) (map[string]string, error) {
	if value == "" {
		return nil, nil
	}
	labels := make(map[string]string)
	for _, item := range strings.Split(value, ",") {
		key, labelValue, found := strings.Cut(item, "=")
		if !found {
			return nil, fmt.Errorf("unexpected label spec: %s", item)
		}
		if key == "" {
			return nil, errors.New("unexpected empty label key")
		}
		labels[key] = labelValue
	}
	return labels, nil
}

func parseRunEnvironment(values []string) ([]v1.EnvVar, error) {
	environment := make([]v1.EnvVar, 0, len(values))
	for _, item := range values {
		name, value, found := strings.Cut(item, "=")
		if !found || name == "" || len(validation.IsEnvVarName(name)) != 0 {
			return nil, fmt.Errorf("invalid env: %v", item)
		}
		environment = append(environment, v1.EnvVar{
			Name:  name,
			Value: value,
		})
	}
	return environment, nil
}

func runRestartPolicy(value string, interactive bool) (v1.RestartPolicy, error) {
	if value == "" {
		if interactive {
			return v1.RestartPolicyOnFailure, nil
		}
		return v1.RestartPolicyAlways, nil
	}
	policy := v1.RestartPolicy(value)
	switch policy {
	case v1.RestartPolicyAlways, v1.RestartPolicyNever, v1.RestartPolicyOnFailure:
		return policy, nil
	default:
		return "", fmt.Errorf("invalid restart policy: %s, valid values are: %s, %s, %s", value, v1.RestartPolicyAlways, v1.RestartPolicyOnFailure, v1.RestartPolicyNever)
	}
}

func runImagePullPolicy(value string) (v1.PullPolicy, error) {
	policy := v1.PullPolicy(value)
	switch policy {
	case v1.PullAlways, v1.PullIfNotPresent, v1.PullNever, "":
		return policy, nil
	default:
		return "", fmt.Errorf("invalid image pull policy: %s, valid values are: %s, %s, %s", value, v1.PullAlways, v1.PullIfNotPresent, v1.PullNever)
	}
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
		return runCodeExitError(message, int(terminated.ExitCode))
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
			initial.SinceTime = runTimestampNow()
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
