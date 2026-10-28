package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type kustomizeConfig struct {
	Stdout               io.Writer
	Arguments            []string
	AsCurrentUser        bool
	EnableAlphaPlugins   bool
	EnableHelm           bool
	EnableManagedbyLabel bool
	Env                  []string
	HelmAPIVersions      []string
	HelmCommand          string
	HelmDebug            bool
	HelmKubeVersion      string
	LoadRestrictor       string
	Mount                []string
	Network              bool
	NetworkName          string
	Output               string
	Reorder              string
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

		arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
		cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[kustomizeConfig]) {
			form.Sequence(arguments, "DIR", cli.Count.ZeroOrMore())
		})

		asCurrentUserFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AsCurrentUser))
		asCurrentUserFlag.Description("use the uid and gid of the command executor to run the function in the container").Long("as-current-user").ImplicitValue(true)

		enableAlphaPluginsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.EnableAlphaPlugins))
		enableAlphaPluginsFlag.Description("enable kustomize plugins").Long("enable-alpha-plugins").ImplicitValue(true)

		enableHelmFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.EnableHelm))
		enableHelmFlag.Description("Enable use of the Helm chart inflator generator.").Long("enable-helm").ImplicitValue(true)

		enableManagedbyLabelFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.EnableManagedbyLabel))
		enableManagedbyLabelFlag = enableManagedbyLabelFlag.Description("enable adding app.kubernetes.io/managed-by").Hidden().Release(preexistingDeprecation("The flag `enable-managedby-label` has been deprecated. Use the `managedByLabel` option in the `buildMetadata` field instead."))
		enableManagedbyLabelFlag.Long("enable-managedby-label").ImplicitValue(true)

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

		loadRestrictorFlag := cb.Flags.String(&cb.Config.LoadRestrictor)
		loadRestrictorFlag.Description("if set to 'LoadRestrictionsNone', local kustomizations may load files from outside their root. This does, however, break the relocatability of the kustomization.").Long("load-restrictor")
		cb.Values.Default(loadRestrictorFlag, "LoadRestrictionsRootOnly")

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

		reorderFlag := cb.Flags.String(&cb.Config.Reorder)
		reorderFlag = reorderFlag.Description("Reorder the resources just before output. Use 'legacy' to apply a legacy reordering (Namespaces first, Webhooks last, etc). Use 'none' to suppress a final reordering.").Hidden().Release(preexistingDeprecation("use the new 'sortOptions' field in kustomization.yaml instead."))
		reorderFlag.Long("reorder")
		cb.Values.Default(reorderFlag, "legacy")
	},
	Run: runKustomize,
}
