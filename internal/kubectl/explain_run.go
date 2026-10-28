/*
Copyright 2014 The Kubernetes Authors.

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

package kubectl

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/explain/v2"
)

func runExplain(_ context.Context, cfg *explainConfig) error {
	factory := cfg.Client.factory()
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		return err
	}
	openAPIV3Client, err := factory.OpenAPIV3Client()
	if err != nil {
		return err
	}

	var resource schema.GroupVersionResource
	var fields []string
	if cfg.APIVersion.Version == "" {
		resource, fields, err = explain.SplitAndParseResourceRequestWithMatchingPrefix(cfg.Resource, mapper)
	} else {
		resource, fields, err = explain.SplitAndParseResourceRequest(cfg.Resource, mapper)
	}
	if err != nil {
		return err
	}

	if cfg.Output == "plaintext-openapiv2" {
		return renderExplainOpenAPIV2(cfg, factory, mapper, resource, fields)
	}
	if _, err := openAPIV3Client.Paths(); err != nil {
		return renderExplainOpenAPIV2(cfg, factory, mapper, resource, fields)
	}
	if cfg.APIVersion.Version != "" {
		resource.Group = cfg.APIVersion.Group
		resource.Version = cfg.APIVersion.Version
	}
	return v2.PrintModelDescription(
		fields,
		cfg.Streams.Out,
		openAPIV3Client,
		resource,
		cfg.Recursive,
		cfg.MaxDepth,
		cfg.Output,
	)
}

func renderExplainOpenAPIV2(cfg *explainConfig, factory util.Factory, mapper meta.RESTMapper, resource schema.GroupVersionResource, fields []string) error {
	kind, _ := mapper.KindFor(resource)
	if kind.Empty() {
		var err error
		kind, err = mapper.KindFor(resource.GroupResource().WithVersion(""))
		if err != nil {
			return err
		}
	}
	if cfg.APIVersion.Version != "" {
		kind = cfg.APIVersion.WithKind(kind.Kind)
	}

	resources, err := factory.OpenAPISchema()
	if err != nil {
		return err
	}
	model := resources.LookupResource(kind)
	if model == nil {
		return fmt.Errorf("couldn't find resource for %q", kind)
	}
	return explain.PrintModelDescription(fields, cfg.Streams.Out, model, kind, cfg.Recursive, cfg.MaxDepth)
}
