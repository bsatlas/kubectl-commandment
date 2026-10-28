package kubectl

import (
	"context"

	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/describe"
)

func runDescribe(_ context.Context, cfg *describeConfig) error {
	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	if cfg.AllNamespaces {
		enforceNamespace = false
	}

	builderArguments := append([]string(nil), cfg.QualifiedResources...)
	if cfg.ResourceType != "" {
		builderArguments = make([]string, 0, 1+len(cfg.Names))
		builderArguments = append(builderArguments, cfg.ResourceType)
		builderArguments = append(builderArguments, cfg.Names...)
	}

	filenameOptions := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	options := describe.DescribeOptions{
		Selector:                cfg.Selector,
		Namespace:               namespace,
		Describer:               describeResourceDescriber(factory),
		NewBuilder:              factory.NewBuilder,
		BuilderArgs:             builderArguments,
		EnforceNamespace:        enforceNamespace,
		AllNamespaces:           cfg.AllNamespaces,
		ShowEventsExplicitlySet: explicitFlagValue(cfg.ShowEventsState),
		DescriberSettings:       describeSettings(cfg),
		FilenameOptions:         &filenameOptions,
		IOStreams:               cfg.Streams,
	}
	return options.Run()
}
