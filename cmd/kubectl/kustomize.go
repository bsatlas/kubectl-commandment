package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
)

type kustomizeConfig struct {
	Stdout               io.Writer
	Directory            string
	AsCurrentUser        bool
	EnableAlphaPlugins   bool
	EnableHelm           bool
	EnableManagedbyLabel bool
	ManagedbyLabelEnv    bool
	Env                  []string
	HelmAPIVersions      []string
	HelmCommand          string
	HelmDebug            bool
	HelmKubeVersion      string
	LoadRestrictor       types.LoadRestrictions
	Mount                []string
	Network              bool
	NetworkName          string
	Output               string
	Reorder              krusty.ReorderOption
	ReorderState         cli.OptionalValue[cli.ValueState]
}

var managedbyLabelEnvironmentLanguage = cli.ValueCodec[bool]{
	Name: "kustomize-managedby-label-environment",
	Parse: func(value string) (bool, error) {
		return value == "on", nil
	},
	Format:   strconv.FormatBool,
	Snapshot: cli.Value.Assign[bool](),
}

var loadRestrictionsLanguage = cli.ValueCodec[types.LoadRestrictions]{
	Name: "kustomize-load-restrictor",
	Parse: func(value string) (types.LoadRestrictions, error) {
		switch value {
		case "", types.LoadRestrictionsRootOnly.String():
			return types.LoadRestrictionsRootOnly, nil
		case types.LoadRestrictionsNone.String():
			return types.LoadRestrictionsNone, nil
		default:
			return 0, cli.Rejection.Message(
				"load-restrictor",
				fmt.Sprintf("illegal flag value --load-restrictor %s; legal values: [%s %s]", value, types.LoadRestrictionsRootOnly.String(), types.LoadRestrictionsNone.String()),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
	},
	Format:   func(value types.LoadRestrictions) string { return value.String() },
	Snapshot: cli.Value.Assign[types.LoadRestrictions](),
}

var reorderLanguage = cli.ValueCodec[krusty.ReorderOption]{
	Name: "kustomize-reorder",
	Parse: func(value string) (krusty.ReorderOption, error) {
		switch value {
		case string(krusty.ReorderOptionLegacy):
			return krusty.ReorderOptionLegacy, nil
		case string(krusty.ReorderOptionNone):
			return krusty.ReorderOptionNone, nil
		default:
			return "", cli.Rejection.Message(
				"reorder",
				fmt.Sprintf("illegal flag value --reorder %s; legal values: [%s %s]", value, krusty.ReorderOptionLegacy, krusty.ReorderOptionNone),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
	},
	Format:   func(value krusty.ReorderOption) string { return string(value) },
	Snapshot: cli.Value.Assign[krusty.ReorderOption](),
}

var kustomizeCommand = cli.Command[kustomizeConfig]{
	Name: "kustomize",
	Description: cli.Description{
		Short: "Build a kustomization target from a directory or URL",
		Long:  "Build a set of KRM resources using a 'kustomization.yaml' file. The DIR argument must be a path to a directory containing 'kustomization.yaml', or a git repository URL with a path suffix specifying same with respect to the repository root. If DIR is omitted, '.' is assumed.",
	},
	Examples: []cli.Example{
		{
			Description: "Build the current working directory",
			CommandLine: "kubectl kustomize",
		},
		{
			Description: "Build some shared configuration directory",
			CommandLine: "kubectl kustomize /home/config/production",
		},
		{
			Description: "Build from github",
			CommandLine: "kubectl kustomize https://github.com/kubernetes-sigs/kustomize.git/examples/helloWorld?ref=v1.0.6",
		},
	},
	Declare: func(cb *cli.CommandBuilder[kustomizeConfig]) {
		cb.Access.Stdout(&cb.Config.Stdout)

		directory := cb.Values.String(&cb.Config.Directory)
		cb.Arguments.ValueFor(directory, "DIR").Optional()

		asCurrentUserFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AsCurrentUser))
		asCurrentUserFlag.Description("use the uid and gid of the command executor to run the function in the container").Long("as-current-user").ImplicitValue(true)

		enableAlphaPluginsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.EnableAlphaPlugins))
		enableAlphaPluginsFlag.Description("enable kustomize plugins").Long("enable-alpha-plugins").ImplicitValue(true)

		enableHelmFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.EnableHelm))
		enableHelmFlag.Description("Enable use of the Helm chart inflator generator.").Long("enable-helm").ImplicitValue(true)

		enableManagedbyLabelFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.EnableManagedbyLabel))
		enableManagedbyLabelFlag.Description("enable adding app.kubernetes.io/managed-by").Hidden().Release(preexistingDeprecation("The flag `enable-managedby-label` has been deprecated. Use the `managedByLabel` option in the `buildMetadata` field instead."))
		enableManagedbyLabelFlag.Long("enable-managedby-label").ImplicitValue(true)

		managedbyLabelEnv := cb.Values.Value(&cb.Config.ManagedbyLabelEnv, managedbyLabelEnvironmentLanguage)
		cb.Values.Environment(managedbyLabelEnv, konfig.EnableManagedbyLabelEnv)

		envValue := cb.Values.Sequence(&cb.Config.Env, cli.Value.String())
		envFlag := cb.Flags.Sequence(envValue)
		envFlag.Description("a list of environment variables to be used by functions").Long("env")
		envFlag.Short('e')

		helmAPIVersionsValue := cb.Values.Sequence(&cb.Config.HelmAPIVersions, cli.Value.String())
		helmAPIVersionsFlag := cb.Flags.Sequence(helmAPIVersionsValue)
		helmAPIVersionsFlag.Description("Kubernetes api versions used by Helm for Capabilities.APIVersions").Long("helm-api-versions")

		helmCommandFlag := cb.Flags.String(&cb.Config.HelmCommand)
		helmCommandFlag.Description("helm command (path to executable)").Long("helm-command")
		cb.Values.Default(helmCommandFlag, "helm")

		helmDebugFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.HelmDebug))
		helmDebugFlag.Description("Enable debug output from the Helm chart inflator generator.").Long("helm-debug").ImplicitValue(true)

		helmKubeVersionFlag := cb.Flags.String(&cb.Config.HelmKubeVersion)
		helmKubeVersionFlag.Description("Kubernetes version used by Helm for Capabilities.KubeVersion").Long("helm-kube-version")

		loadRestrictor := cb.Values.Value(&cb.Config.LoadRestrictor, loadRestrictionsLanguage)
		loadRestrictorFlag := cb.Flags.ValueFor(loadRestrictor)
		loadRestrictorFlag.Description("if set to 'LoadRestrictionsNone', local kustomizations may load files from outside their root. This does, however, break the relocatability of the kustomization.").Long("load-restrictor")
		cb.Values.Default(loadRestrictorFlag, types.LoadRestrictionsRootOnly)

		mountValue := cb.Values.Sequence(&cb.Config.Mount, cli.Value.String())
		mountFlag := cb.Flags.Sequence(mountValue)
		mountFlag.Description("a list of storage options read from the filesystem").Long("mount")

		networkFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Network))
		networkFlag.Description("enable network access for functions that declare it").Long("network").ImplicitValue(true)

		networkNameFlag := cb.Flags.String(&cb.Config.NetworkName)
		networkNameFlag.Description("the docker network to run the container in").Long("network-name")
		cb.Values.Default(networkNameFlag, "bridge")

		outputFlag := cb.Flags.String(&cb.Config.Output)
		outputFlag.Description("If specified, write output to this path.").Long("output")
		outputFlag.Short('o')

		reorder := cb.Values.Value(&cb.Config.Reorder, reorderLanguage)
		reorderFlag := cb.Flags.ValueFor(reorder)
		reorderFlag.Description("Reorder the resources just before output. Use 'legacy' to apply a legacy reordering (Namespaces first, Webhooks last, etc). Use 'none' to suppress a final reordering.").Hidden().Release(preexistingDeprecation("use the new 'sortOptions' field in kustomization.yaml instead."))
		reorderFlag.Long("reorder")
		cb.Values.Default(reorderFlag, krusty.ReorderOptionLegacy)
		cb.Access.ValueState(reorder, &cb.Config.ReorderState)
	},
	Run: runKustomize,
}
