package main

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func debugObjectMeta(name, namespace string, labels, annotations map[string]string) v1.ObjectMeta {
	metadata := v1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: annotations,
	}
	return metadata
}

func debugCreateOptions() v1.CreateOptions {
	return v1.CreateOptions{}
}

func debugPatchOptions() v1.PatchOptions {
	return v1.PatchOptions{}
}

func debugDeleteOptionsNow() v1.DeleteOptions {
	return *v1.NewDeleteOptions(0)
}

func debugEphemeralContainersDisabled(err error) bool {
	status, ok := err.(*errors.StatusError)
	return ok && status.Status().Reason == v1.StatusReasonNotFound && status.ErrStatus.Details.Name == ""
}

func debugPodListWatch(client kubernetes.Interface, namespace, podName string, ctx context.Context) *cache.ListWatch {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", podName).String()
	listWatch := cache.ListWatch{
		ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return client.CoreV1().Pods(namespace).List(ctx, options)
		},
		WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return client.CoreV1().Pods(namespace).Watch(ctx, options)
		},
	}
	return &listWatch
}
