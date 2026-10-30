package main

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/interrupt"
)

var errRolloutStatusObjectDeleted = errors.New("object has been deleted")

type rolloutStatusObjectTypeError struct {
	actual string
}

func (err rolloutStatusObjectTypeError) Error() string {
	return "internal error: rollout status object has type " + err.actual
}

type rolloutStatusEventError struct {
	event watch.Event
}

func (err rolloutStatusEventError) Error() string {
	return fmt.Sprintf("internal error: unexpected event %#v", err.event)
}

func runRolloutStatus(ctx context.Context, cfg *rolloutStatusConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		return err
	}

	filenameOptions := resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(namespace).DefaultNamespace().
		LabelSelectorParam(cfg.Selector)
	addFilenameSources(builder, enforceNamespace, filenameOptions, cfg.Streams.In)
	result := builder.
		ResourceTypeOrNameArgs(true, cfg.Resources.Args()...).
		ContinueOnError().
		Latest().
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	resourceFound := false
	err = result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		resourceFound = true
		statusViewer, err := polymorphichelpers.StatusViewerFn(info.ResourceMapping())
		if err != nil {
			return err
		}

		fieldSelector := fields.OneTermEqualSelector("metadata.name", info.Name).String()
		watchCtx, cancel := watchtools.ContextWithOptionalTimeout(ctx, cfg.Timeout)
		defer cancel()
		listWatcher := cache.ToListWatcherWithWatchListSemantics(&cache.ListWatch{
			ListWithContextFunc: func(requestCtx context.Context, options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector
				return dynamicClient.Resource(info.Mapping.Resource).Namespace(info.Namespace).List(requestCtx, options)
			},
			WatchFuncWithContext: func(requestCtx context.Context, options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fieldSelector
				return dynamicClient.Resource(info.Mapping.Resource).Namespace(info.Namespace).Watch(requestCtx, options)
			},
		}, dynamicClient)

		intr := interrupt.New(nil, cancel)
		return intr.Run(func() error {
			_, err := watchtools.UntilWithSync(watchCtx, listWatcher, &unstructured.Unstructured{}, nil, func(event watch.Event) (bool, error) {
				switch event.Type {
				case watch.Added, watch.Modified:
					object, ok := event.Object.(runtime.Unstructured)
					if !ok {
						objectTypeError := rolloutStatusObjectTypeError{actual: fmt.Sprintf("%T", event.Object)}
						return true, objectTypeError
					}
					status, done, err := statusViewer.Status(object, cfg.Revision)
					if err != nil {
						return false, err
					}
					if _, err := fmt.Fprint(cfg.Streams.Out, status); err != nil {
						return true, err
					}
					return done || !cfg.Watch, nil
				case watch.Deleted:
					return true, errRolloutStatusObjectDeleted
				default:
					eventError := rolloutStatusEventError{event: event}
					return true, eventError
				}
			})
			return err
		})
	})
	if err != nil {
		return err
	}
	if !resourceFound {
		_, err := fmt.Fprintf(cfg.Streams.ErrOut, "No resources found in %s namespace.\n", namespace)
		return err
	}
	return nil
}
