/*
Copyright 2020 The Kubernetes Authors.

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
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/attach"
	"k8s.io/kubectl/pkg/cmd/debug"
	"k8s.io/kubectl/pkg/cmd/exec"
	"k8s.io/kubectl/pkg/cmd/logs"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/interrupt"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

type debugOperation struct {
	Args                  []string
	ArgsOnly              bool
	Attach                bool
	Container             string
	CopyTo                string
	Replace               bool
	Env                   []v1.EnvVar
	Image                 string
	Interactive           bool
	KeepLabels            bool
	KeepAnnotations       bool
	KeepLiveness          bool
	KeepReadiness         bool
	KeepStartup           bool
	KeepInitContainers    bool
	Namespace             string
	ExplicitNamespace     bool
	TargetNames           []string
	PullPolicy            v1.PullPolicy
	Quiet                 bool
	SameNode              bool
	SetImages             map[string]string
	ShareProcesses        bool
	ShareProcessesChanged bool
	TargetContainer       string
	TTY                   bool
	Profile               string
	CustomProfileFile     string
	CustomProfile         *v1.Container
	Applier               debug.ProfileApplier
	Client                kubernetes.Interface
	Builder               *resource.Builder
	Streams               genericiooptions.IOStreams
	WarningPrinter        *printers.WarningPrinter
	Filename              resource.FilenameOptions
	RestClientGetter      genericclioptions.RESTClientGetter
}

var debugNameSuffix = rand.String

func runDebug(ctx context.Context, cfg *debugConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, explicitNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	env := make([]v1.EnvVar, 0, len(cfg.Env))
	names := make([]string, 0, len(cfg.Env))
	for name := range cfg.Env {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		env = append(env, v1.EnvVar{
			Name:  name,
			Value: cfg.Env[name],
		})
	}

	keep := debug.KeepFlags{
		Labels:         cfg.KeepLabels,
		Annotations:    cfg.KeepAnnotations,
		Liveness:       cfg.KeepLiveness,
		Readiness:      cfg.KeepReadiness,
		Startup:        cfg.KeepStartup,
		InitContainers: cfg.KeepInitContainers,
	}
	applier, err := debug.NewProfileApplier(cfg.Profile, keep)
	if err != nil {
		return err
	}

	operation := debugOperation{
		Args:                  slices.Clone(cfg.Arguments),
		ArgsOnly:              cfg.ArgumentsOnly,
		Attach:                cfg.Attach,
		Container:             cfg.Container,
		CopyTo:                cfg.CopyTo,
		Replace:               cfg.Replace,
		Env:                   env,
		Image:                 cfg.Image,
		Interactive:           cfg.Stdin,
		KeepLabels:            cfg.KeepLabels,
		KeepAnnotations:       cfg.KeepAnnotations,
		KeepLiveness:          cfg.KeepLiveness,
		KeepReadiness:         cfg.KeepReadiness,
		KeepStartup:           cfg.KeepStartup,
		KeepInitContainers:    cfg.KeepInitContainers,
		Namespace:             namespace,
		ExplicitNamespace:     explicitNamespace,
		TargetNames:           slices.Clone(cfg.TargetNames),
		PullPolicy:            cfg.ImagePullPolicy,
		Quiet:                 cfg.Quiet,
		SameNode:              cfg.SameNode,
		SetImages:             cfg.SetImage,
		ShareProcesses:        cfg.ShareProcesses,
		ShareProcessesChanged: explicitFlagValue(cfg.ShareProcessesState),
		TargetContainer:       cfg.Target,
		TTY:                   cfg.TTY,
		Profile:               cfg.Profile,
		CustomProfileFile:     cfg.Custom,
		Applier:               applier,
		Client:                client,
		Builder:               resource.NewBuilder(factory),
		Streams:               cfg.Streams,
		WarningPrinter: printers.NewWarningPrinter(
			cfg.Streams.ErrOut,
			printers.WarningPrinterOptions{
				Color: printers.AllowsColorOutput(cfg.Streams.ErrOut),
			},
		),
		Filename: resource.FilenameOptions{
			Filenames: slices.Concat(cfg.Filename...),
		},
		RestClientGetter: factory,
	}

	if operation.CustomProfileFile != "" {
		custom, err := os.ReadFile(operation.CustomProfileFile)
		if err != nil {
			return fmt.Errorf("must pass a container spec json file for custom profile: %w", err)
		}
		container := new(v1.Container)
		if err := json.Unmarshal(custom, container); err != nil {
			if err := yaml.Unmarshal(custom, container); err != nil {
				return fmt.Errorf("%s does not contain a valid container spec: %w", operation.CustomProfileFile, err)
			}
		}
		operation.CustomProfile = container
	}

	if operation.TargetContainer != "" && !operation.Quiet {
		_, _ = fmt.Fprintf(operation.Streams.Out, "Targeting container %q. If you don't see processes from this container it may be because the container runtime doesn't support this feature.\n", operation.TargetContainer)
	}
	if operation.CustomProfile != nil {
		if operation.CustomProfile.Name != "" || len(operation.CustomProfile.Command) > 0 || operation.CustomProfile.Image != "" || operation.CustomProfile.Lifecycle != nil || len(operation.CustomProfile.VolumeDevices) > 0 {
			return fmt.Errorf("name, command, image, lifecycle and volume devices are not modifiable via custom profile")
		}
	}
	return operation.run(ctx)
}

func (o *debugOperation) run(ctx context.Context) error {
	builder := o.Builder.
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...)
	addFilenameSources(builder, o.ExplicitNamespace, o.Filename, o.Streams.In)
	result := builder.
		NamespaceParam(o.Namespace).DefaultNamespace().ResourceNames("pods", o.TargetNames...).
		Do()
	if err := result.Err(); err != nil {
		return err
	}
	return result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		var debugPod *v1.Pod
		var containerName string
		var visitErr error
		switch object := info.Object.(type) {
		case *v1.Node:
			debugPod, containerName, visitErr = o.visitNode(ctx, object)
		case *v1.Pod:
			debugPod, containerName, visitErr = o.visitPod(ctx, object)
		default:
			visitErr = fmt.Errorf("%q not supported by debug", info.Mapping.GroupVersionKind)
		}
		if visitErr != nil {
			return visitErr
		}
		if o.Attach && containerName != "" {
			return o.handleAttachPod(ctx, o.RestClientGetter, "kubectl", debugPod.Namespace, debugPod.Name, containerName)
		}
		return nil
	})
}

func (o *debugOperation) visitNode(ctx context.Context, node *v1.Node) (*v1.Pod, string, error) {
	pods := o.Client.CoreV1().Pods(o.Namespace)
	debugPod, err := o.generateNodeDebugPod(node)
	if err != nil {
		return nil, "", err
	}
	newPod, err := pods.Create(ctx, debugPod, metav1.CreateOptions{})
	if err != nil {
		return nil, "", err
	}

	return newPod, newPod.Spec.Containers[0].Name, nil
}

// visitPod handles debugging for pod targets by (depending on options):
//  1. Creating an ephemeral debug container in an existing pod, OR
//  2. Making a copy of pod with certain attributes changed
//
// visitPod returns a pod and debug container name for subsequent attach, if applicable.
func (o *debugOperation) visitPod(ctx context.Context, pod *v1.Pod) (*v1.Pod, string, error) {
	if len(o.CopyTo) > 0 {
		return o.debugByCopy(ctx, pod)
	}
	return o.debugByEphemeralContainer(ctx, pod)
}

// debugByEphemeralContainer runs an EphemeralContainer in the target Pod for use as a debug container
func (o *debugOperation) debugByEphemeralContainer(ctx context.Context, pod *v1.Pod) (*v1.Pod, string, error) {
	klog.V(2).Infof("existing ephemeral containers: %v", pod.Spec.EphemeralContainers)
	podJS, err := json.Marshal(pod)
	if err != nil {
		return nil, "", fmt.Errorf("error creating JSON for pod: %v", err)
	}

	debugPod, debugContainer, err := o.generateDebugContainer(pod)
	if err != nil {
		return nil, "", err
	}
	klog.V(2).Infof("new ephemeral container: %#v", debugContainer)

	o.displayWarning((*v1.Container)(&debugContainer.EphemeralContainerCommon), pod)

	debugJS, err := json.Marshal(debugPod)
	if err != nil {
		return nil, "", fmt.Errorf("error creating JSON for debug container: %v", err)
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(podJS, debugJS, pod)
	if err != nil {
		return nil, "", fmt.Errorf("error creating patch to add debug container: %v", err)
	}
	klog.V(2).Infof("generated strategic merge patch for debug container: %s", patch)

	pods := o.Client.CoreV1().Pods(pod.Namespace)
	result, err := pods.Patch(ctx, pod.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "ephemeralcontainers")
	if err != nil {
		// The apiserver will return a 404 when the EphemeralContainers feature is disabled because the `/ephemeralcontainers` subresource
		// is missing. Unlike the 404 returned by a missing pod, the status details will be empty.
		status, ok := err.(*errors.StatusError)
		if ok && status.Status().Reason == metav1.StatusReasonNotFound && status.ErrStatus.Details.Name == "" {
			return nil, "", fmt.Errorf("ephemeral containers are disabled for this cluster (error from server: %q)", err)
		}

		return nil, "", err
	}

	return result, debugContainer.Name, nil
}

// applyCustomProfile applies given partial container json file on to the profile
// incorporated debug pod.
func (o *debugOperation) applyCustomProfile(debugPod *v1.Pod, containerName string) error {
	o.CustomProfile.Name = containerName
	customJS, err := json.Marshal(o.CustomProfile)
	if err != nil {
		return fmt.Errorf("unable to marshall custom profile: %w", err)
	}

	var index int
	found := false
	for i, val := range debugPod.Spec.Containers {
		if val.Name == containerName {
			index = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("unable to find the %s container in the pod %s", containerName, debugPod.Name)
	}

	var debugContainerJS []byte
	debugContainerJS, err = json.Marshal(debugPod.Spec.Containers[index])
	if err != nil {
		return fmt.Errorf("unable to marshall container: %w", err)
	}

	patchedContainer, err := strategicpatch.StrategicMergePatch(debugContainerJS, customJS, v1.Container{})
	if err != nil {
		return fmt.Errorf("error creating three way patch to add debug container: %w", err)
	}

	err = json.Unmarshal(patchedContainer, &debugPod.Spec.Containers[index])
	if err != nil {
		return fmt.Errorf("unable to unmarshall patched container to container: %w", err)
	}

	return nil
}

// applyCustomProfileEphemeral applies given partial container json file on to the profile
// incorporated ephemeral container of the pod.
func (o *debugOperation) applyCustomProfileEphemeral(debugPod *v1.Pod, containerName string) error {
	o.CustomProfile.Name = containerName
	customJS, err := json.Marshal(o.CustomProfile)
	if err != nil {
		return fmt.Errorf("unable to marshall custom profile: %w", err)
	}

	var index int
	found := false
	for i, val := range debugPod.Spec.EphemeralContainers {
		if val.Name == containerName {
			index = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("unable to find the %s ephemeral container in the pod %s", containerName, debugPod.Name)
	}

	var debugContainerJS []byte
	debugContainerJS, err = json.Marshal(debugPod.Spec.EphemeralContainers[index])
	if err != nil {
		return fmt.Errorf("unable to marshall ephemeral container:%w", err)
	}

	patchedContainer, err := strategicpatch.StrategicMergePatch(debugContainerJS, customJS, v1.Container{})
	if err != nil {
		return fmt.Errorf("error creating three way patch to add debug container: %w", err)
	}

	err = json.Unmarshal(patchedContainer, &debugPod.Spec.EphemeralContainers[index])
	if err != nil {
		return fmt.Errorf("unable to unmarshall patched container to ephemeral container: %w", err)
	}

	return nil
}

// debugByCopy runs a copy of the target Pod with a debug container added or an original container modified
func (o *debugOperation) debugByCopy(ctx context.Context, pod *v1.Pod) (*v1.Pod, string, error) {
	copied, dc, err := o.generatePodCopyWithDebugContainer(pod)
	if err != nil {
		return nil, "", err
	}

	var debugContainer *v1.Container
	for i := range copied.Spec.Containers {
		if copied.Spec.Containers[i].Name == dc {
			debugContainer = &copied.Spec.Containers[i]
			break
		}
	}
	o.displayWarning(debugContainer, copied)

	created, err := o.Client.CoreV1().Pods(copied.Namespace).Create(ctx, copied, metav1.CreateOptions{})
	if err != nil {
		return nil, "", err
	}
	if o.Replace {
		err := o.Client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, *metav1.NewDeleteOptions(0))
		if err != nil {
			return nil, "", err
		}
	}
	return created, dc, nil
}

// Display warning message if some capabilities are set by profile and non-root user is specified in .Spec.SecurityContext.RunAsUser.(#1650)
func (o *debugOperation) displayWarning(container *v1.Container, pod *v1.Pod) {
	if container == nil {
		return
	}

	if pod.Spec.SecurityContext.RunAsUser == nil || *pod.Spec.SecurityContext.RunAsUser == 0 {
		return
	}

	if container.SecurityContext == nil {
		return
	}

	if container.SecurityContext.RunAsUser != nil && *container.SecurityContext.RunAsUser == 0 {
		return
	}

	if (container.SecurityContext.Privileged == nil || !*container.SecurityContext.Privileged) &&
		(container.SecurityContext.Capabilities == nil || len(container.SecurityContext.Capabilities.Add) == 0) {
		return
	}

	_, _ = fmt.Fprintln(o.Streams.ErrOut, `Warning: Non-root user is configured for the entire target Pod, and some capabilities granted by debug profile may not work. Please consider using "--custom" with a custom profile that specifies "securityContext.runAsUser: 0".`)
}

// generateDebugContainer returns a debugging pod and an EphemeralContainer suitable for use as a debug container
// in the given pod.
func (o *debugOperation) generateDebugContainer(pod *v1.Pod) (*v1.Pod, *v1.EphemeralContainer, error) {
	name := o.computeDebugContainerName(pod)
	ec := &v1.EphemeralContainer{
		EphemeralContainerCommon: v1.EphemeralContainerCommon{
			Name:                     name,
			Env:                      o.Env,
			Image:                    o.Image,
			ImagePullPolicy:          o.PullPolicy,
			Stdin:                    o.Interactive,
			TerminationMessagePolicy: v1.TerminationMessageReadFile,
			TTY:                      o.TTY,
		},
		TargetContainerName: o.TargetContainer,
	}

	if o.ArgsOnly {
		ec.Args = o.Args
	} else {
		ec.Command = o.Args
	}

	copied := pod.DeepCopy()
	copied.Spec.EphemeralContainers = append(copied.Spec.EphemeralContainers, *ec)
	if err := o.Applier.Apply(copied, name, copied); err != nil {
		return nil, nil, err
	}

	if o.CustomProfile != nil {
		err := o.applyCustomProfileEphemeral(copied, ec.Name)
		if err != nil {
			return nil, nil, err
		}
	}

	ec = &copied.Spec.EphemeralContainers[len(copied.Spec.EphemeralContainers)-1]

	return copied, ec, nil
}

// generateNodeDebugPod generates a debugging pod that schedules on the specified node.
// The generated pod will run in the host PID, Network & IPC namespaces, and it will have the node's filesystem mounted at /host.
func (o *debugOperation) generateNodeDebugPod(node *v1.Node) (*v1.Pod, error) {
	cn := "debugger"
	// Setting a user-specified container name doesn't make much difference when there's only one container,
	// but the argument exists for pod debugging so it might be confusing if it didn't work here.
	if len(o.Container) > 0 {
		cn = o.Container
	}

	// The name of the debugging pod is based on the target node, and it's not configurable to
	// limit the number of command line flags. There may be a collision on the name, but this
	// should be rare enough that it's not worth the API round trip to check.
	pn := fmt.Sprintf("node-debugger-%s-%s", node.Name, debugNameSuffix(5))
	if !o.Quiet {
		fmt.Fprintf(o.Streams.Out, "Creating debugging pod %s with container %s on node %s.\n", pn, cn, node.Name)
	}

	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: pn,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kubectl-debug",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:                     cn,
					Env:                      o.Env,
					Image:                    o.Image,
					ImagePullPolicy:          o.PullPolicy,
					Stdin:                    o.Interactive,
					TerminationMessagePolicy: v1.TerminationMessageReadFile,
					TTY:                      o.TTY,
				},
			},
			NodeName:      node.Name,
			RestartPolicy: v1.RestartPolicyNever,
			Tolerations: []v1.Toleration{
				{
					Operator: v1.TolerationOpExists,
				},
			},
		},
	}

	if o.ArgsOnly {
		p.Spec.Containers[0].Args = o.Args
	} else {
		p.Spec.Containers[0].Command = o.Args
	}

	if err := o.Applier.Apply(p, cn, node); err != nil {
		return nil, err
	}

	if o.CustomProfile != nil {
		err := o.applyCustomProfile(p, cn)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

// generatePodCopyWithDebugContainer takes a Pod and returns a copy and the debug container name of that copy
func (o *debugOperation) generatePodCopyWithDebugContainer(pod *v1.Pod) (*v1.Pod, string, error) {
	copied := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        o.CopyTo,
			Namespace:   pod.Namespace,
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
		},
		Spec: *pod.Spec.DeepCopy(),
	}
	// set EphemeralContainers to nil so that the copy of pod can be created
	copied.Spec.EphemeralContainers = nil
	// change ShareProcessNamespace configuration only when commanded explicitly
	if o.ShareProcessesChanged {
		copied.Spec.ShareProcessNamespace = ptr.To(o.ShareProcesses)
	}
	if !o.SameNode {
		copied.Spec.NodeName = ""
	}

	// Apply image mutations
	for i, c := range copied.Spec.Containers {
		override := o.SetImages["*"]
		if img, ok := o.SetImages[c.Name]; ok {
			override = img
		}
		if len(override) > 0 {
			copied.Spec.Containers[i].Image = override
		}
	}

	name, containerByName := o.Container, debugContainerNameToRef(copied)

	c, ok := containerByName[name]
	if !ok {
		// Adding a new debug container
		if len(o.Image) == 0 {
			if len(o.SetImages) > 0 {
				// This was a --set-image only invocation
				return copied, "", nil
			}
			return nil, "", fmt.Errorf("you must specify image when creating new container")
		}

		if len(name) == 0 {
			name = o.computeDebugContainerName(copied)
		}
		copied.Spec.Containers = append(copied.Spec.Containers, v1.Container{
			Name:                     name,
			TerminationMessagePolicy: v1.TerminationMessageReadFile,
		})
		c = &copied.Spec.Containers[len(copied.Spec.Containers)-1]
	}

	if len(o.Args) > 0 {
		if o.ArgsOnly {
			c.Args = o.Args
		} else {
			c.Command = o.Args
			c.Args = nil
		}
	}
	if len(o.Env) > 0 {
		c.Env = o.Env
	}
	if len(o.Image) > 0 {
		c.Image = o.Image
	}
	if len(o.PullPolicy) > 0 {
		c.ImagePullPolicy = o.PullPolicy
	}
	c.Stdin = o.Interactive
	c.TTY = o.TTY

	err := o.Applier.Apply(copied, c.Name, pod)
	if err != nil {
		return nil, "", err
	}

	if o.CustomProfile != nil {
		err = o.applyCustomProfile(copied, name)
		if err != nil {
			return nil, "", err
		}
	}

	return copied, name, nil
}

func (o *debugOperation) computeDebugContainerName(pod *v1.Pod) string {
	if len(o.Container) > 0 {
		return o.Container
	}

	cn, containerByName := "", debugContainerNameToRef(pod)
	for len(cn) == 0 || (containerByName[cn] != nil) {
		cn = fmt.Sprintf("debugger-%s", debugNameSuffix(5))
	}
	if !o.Quiet {
		fmt.Fprintf(o.Streams.Out, "Defaulting debug container name to %s.\n", cn)
	}
	return cn
}

func debugContainerNameToRef(pod *v1.Pod) map[string]*v1.Container {
	names := map[string]*v1.Container{}
	for i := range pod.Spec.Containers {
		ref := &pod.Spec.Containers[i]
		names[ref.Name] = ref
	}
	for i := range pod.Spec.InitContainers {
		ref := &pod.Spec.InitContainers[i]
		names[ref.Name] = ref
	}
	for i := range pod.Spec.EphemeralContainers {
		ref := (*v1.Container)(&pod.Spec.EphemeralContainers[i].EphemeralContainerCommon)
		names[ref.Name] = ref
	}
	return names
}

// waitForContainer watches the given pod until the container is running
func (o *debugOperation) waitForContainer(ctx context.Context, ns, podName, containerName string) (*v1.Pod, error) {
	// TODO: expose the timeout
	ctx, cancel := watchtools.ContextWithOptionalTimeout(ctx, 0*time.Second)
	defer cancel()

	fieldSelector := fields.OneTermEqualSelector("metadata.name", podName).String()
	listWatch := cache.ListWatch{
		ListWithContextFunc: func(listCtx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return o.Client.CoreV1().Pods(ns).List(listCtx, options)
		},
		WatchFuncWithContext: func(watchCtx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return o.Client.CoreV1().Pods(ns).Watch(watchCtx, options)
		},
	}

	intr := interrupt.New(nil, cancel)
	var result *v1.Pod
	err := intr.Run(func() error {
		event, err := watchtools.UntilWithSync(ctx, &listWatch, &v1.Pod{}, nil, func(ev watch.Event) (bool, error) {
			klog.V(2).Infof("watch received event %q with object %T", ev.Type, ev.Object)
			switch ev.Type {
			case watch.Deleted:
				resource := schema.GroupResource{
					Resource: "pods",
				}
				return false, errors.NewNotFound(resource, "")
			}

			p, ok := ev.Object.(*v1.Pod)
			if !ok {
				return false, fmt.Errorf("watch did not return a pod: %v", ev.Object)
			}

			s := debugContainerStatusByName(p, containerName)
			if s == nil {
				return false, nil
			}
			klog.V(2).Infof("debug container status is %v", s)
			if s.State.Running != nil || s.State.Terminated != nil {
				return true, nil
			}
			if !o.Quiet && s.State.Waiting != nil && s.State.Waiting.Message != "" {
				o.WarningPrinter.Print(fmt.Sprintf("container %s: %s", containerName, s.State.Waiting.Message))
			}
			return false, nil
		})
		if event != nil {
			result = event.Object.(*v1.Pod)
		}
		return err
	})

	return result, err
}

func (o *debugOperation) handleAttachPod(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, cmdPath, ns, podName, containerName string) error {
	opts := &attach.AttachOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: o.Streams,
			Stdin:     o.Interactive,
			TTY:       o.TTY,
			Quiet:     o.Quiet,
		},
		CommandName: cmdPath + " attach",

		Attach: &attach.DefaultRemoteAttach{},
	}
	config, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return err
	}
	opts.Config = config
	opts.AttachFunc = attach.DefaultAttachFunc

	pod, err := o.waitForContainer(ctx, ns, podName, containerName)
	if err != nil {
		return err
	}

	opts.Namespace = ns
	opts.Pod = pod
	opts.PodName = podName
	opts.ContainerName = containerName
	if opts.AttachFunc == nil {
		opts.AttachFunc = attach.DefaultAttachFunc
	}

	status := debugContainerStatusByName(pod, containerName)
	if status == nil {
		// impossible path
		return fmt.Errorf("error getting container status of container name %q: %+v", containerName, err)
	}
	if status.State.Terminated != nil {
		klog.V(1).Info("Ephemeral container terminated, falling back to logs")
		return debugLogOptions(ctx, restClientGetter, pod, opts)
	}

	if err := opts.Run(); err != nil {
		fmt.Fprintf(opts.ErrOut, "warning: couldn't attach to pod/%s, falling back to streaming logs: %v\n", podName, err)
		return debugLogOptions(ctx, restClientGetter, pod, opts)
	}
	return nil
}

func debugContainerStatusByName(pod *v1.Pod, containerName string) *v1.ContainerStatus {
	allContainerStatus := [][]v1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses, pod.Status.EphemeralContainerStatuses}
	for _, statusSlice := range allContainerStatus {
		for i := range statusSlice {
			if statusSlice[i].Name == containerName {
				return &statusSlice[i]
			}
		}
	}
	return nil
}

// debugLogOptions logs output from opts to the pods log.
func debugLogOptions(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, pod *v1.Pod, opts *attach.AttachOptions) error {
	ctrName, err := opts.GetContainerName(pod)
	if err != nil {
		return err
	}

	requests, err := polymorphichelpers.LogsForObjectFn(restClientGetter, pod, &v1.PodLogOptions{
		Container: ctrName,
	}, opts.GetPodTimeout, false)
	if err != nil {
		return err
	}
	for _, request := range requests {
		if err := logs.DefaultConsumeRequest(ctx, request, opts.Out); err != nil {
			return err
		}
	}

	return nil
}
