package main

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type exposeLabelSet struct {
	Text   string
	Values map[string]string
}

type exposePort struct {
	Text   string
	Number int32
}

type exposePortSet struct {
	Text   string
	Values []exposePort
}

var exposeLabelSetLanguage = cli.ValueCodec[exposeLabelSet]{
	Name: "kubectl-expose-label-set",
	Parse: func(value string) (exposeLabelSet, error) {
		parsed, err := parseExposeLabelSet(value)
		if err != nil {
			return exposeLabelSet{}, cli.Rejection.Message(
				"expose-label-set",
				err.Error(),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return parsed, nil
	},
	Format: func(value exposeLabelSet) string { return value.Text },
	Snapshot: func(value exposeLabelSet) exposeLabelSet {
		value.Values = maps.Clone(value.Values)
		return value
	},
}

var exposePortSetLanguage = cli.ValueCodec[exposePortSet]{
	Name: "kubectl-expose-port-set",
	Parse: func(value string) (exposePortSet, error) {
		parsed, err := parseExposePortSet(value)
		if err != nil {
			return exposePortSet{}, cli.Rejection.Message(
				"expose-port-set",
				err.Error(),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return parsed, nil
	},
	Format: func(value exposePortSet) string { return value.Text },
	Snapshot: func(value exposePortSet) exposePortSet {
		value.Values = append([]exposePort(nil), value.Values...)
		return value
	},
}

func parseExposeLabelSet(value string) (exposeLabelSet, error) {
	if value == "" {
		return exposeLabelSet{}, nil
	}
	labels := make(map[string]string)
	for item := range strings.SplitSeq(value, ",") {
		parts := strings.Split(item, "=")
		if len(parts) != 2 {
			return exposeLabelSet{}, fmt.Errorf("unexpected label spec: %s", item)
		}
		if parts[0] == "" {
			return exposeLabelSet{}, fmt.Errorf("unexpected empty label key")
		}
		labels[parts[0]] = parts[1]
	}
	result := exposeLabelSet{
		Text:   value,
		Values: labels,
	}
	return result, nil
}

func parseExposePortSet(value string) (exposePortSet, error) {
	if value == "" {
		return exposePortSet{}, nil
	}
	parts := strings.Split(value, ",")
	ports := make([]exposePort, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return exposePortSet{}, err
		}
		ports = append(ports, exposePort{Text: part, Number: int32(number)})
	}
	result := exposePortSet{
		Text:   value,
		Values: ports,
	}
	return result, nil
}

var exposeSessionAffinityLanguage = cli.ValueCodec[v1.ServiceAffinity]{
	Name: "kubectl-expose-session-affinity",
	Parse: func(value string) (v1.ServiceAffinity, error) {
		switch v1.ServiceAffinity(value) {
		case "", v1.ServiceAffinityNone, v1.ServiceAffinityClientIP:
			return v1.ServiceAffinity(value), nil
		default:
			return "", cli.Rejection.Message(
				"session-affinity",
				"--session-affinity must be one of [None ClientIP]",
				cli.Rejection.At(0, 0, len(value)),
			)
		}
	},
	Format:   func(value v1.ServiceAffinity) string { return string(value) },
	Snapshot: func(value v1.ServiceAffinity) v1.ServiceAffinity { return value },
}

type exposeConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	ClusterIP                string
	DryRun                   util.DryRunStrategy
	ExternalIP               string
	FieldManager             string
	Filename                 filenameOptions
	Labels                   exposeLabelSet
	LoadBalancerIP           string
	Name                     string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	OverrideTypeInput        string
	OverrideType             util.OverrideType
	Overrides                string
	Port                     exposePortSet
	Protocol                 string
	Record                   bool
	SaveConfig               bool
	Selector                 exposeLabelSet
	SessionAffinity          v1.ServiceAffinity
	ShowManagedFields        bool
	TargetPort               string
	Template                 string
	Type                     string
}

func exposeCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[exposeConfig] {
	cmd := cli.Command[exposeConfig]{
		Name: "expose",
		Description: cli.Description{
			Short: "Take a replication controller, service, deployment or pod and expose it as a new Kubernetes service",
			Long:  "Expose a resource as a new Kubernetes service.\n\n Looks up a deployment, service, replica set, replication controller or pod by name and uses the selector for that resource as the selector for a new service on the specified port. A deployment or replica set will be exposed as a service only if its selector is convertible to a selector that service supports, i.e. when the selector contains only the matchLabels component. Note that if no port is specified via --port and the exposed resource has multiple ports, all will be re-used by the new service. Also if no labels are specified, the new service will re-use the labels from the resource it exposes.\n\n Possible resources include (case insensitive):\n\n pod (po), service (svc), replicationcontroller (rc), deployment (deploy), replicaset (rs)",
		},
		Examples: []cli.Example{
			{
				Description: "Create a service for a replicated nginx, which serves on port 80 and connects to the containers on port 8000",
				CommandLine: "kubectl expose rc nginx --port=80 --target-port=8000",
			},
			{
				Description: "Create a service for a replication controller identified by type and name specified in \"nginx-controller.yaml\", which serves on port 80 and connects to the containers on port 8000",
				CommandLine: "kubectl expose -f nginx-controller.yaml --port=80 --target-port=8000",
			},
			{
				Description: "Create a service for a pod valid-pod, which serves on port 444 with the name \"frontend\"",
				CommandLine: "kubectl expose pod valid-pod --port=444 --name=frontend",
			},
			{
				Description: "Create a second service based on the above service, exposing the container port 8443 as port 443 with the name \"nginx-https\"",
				CommandLine: "kubectl expose service nginx --port=443 --target-port=8443 --name=nginx-https",
			},
			{
				Description: "Create a service for a replicated streaming application on port 4100 balancing UDP traffic and named 'video-stream'.",
				CommandLine: "kubectl expose rc streamer --port=4100 --protocol=UDP --name=video-stream",
			},
			{
				Description: "Create a service for a replicated nginx using replica set, which serves on port 80 and connects to the containers on port 8000",
				CommandLine: "kubectl expose rs nginx --port=80 --target-port=8000",
			},
			{
				Description: "Create a service for an nginx deployment, which serves on port 80 and connects to the containers on port 8000",
				CommandLine: "kubectl expose deployment nginx --port=80 --target-port=8000",
			},
		},
		Declare: func(cb *cli.CommandBuilder[exposeConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[exposeConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *exposeConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						[]string{"pod", "service", "replicationcontroller", "deployment", "replicaset"},
						resourceArgumentRepeated,
					)).
					From(client)
			})

			declarePrintFlags(cb, &cb.Config.Output, &cb.Config.OutputState, &cb.Config.Template, &cb.Config.AllowMissingTemplateKeys, &cb.Config.ShowManagedFields)

			clusterIPFlag := cb.Flags.String(&cb.Config.ClusterIP)
			cb.Values.ResolveWith(clusterIPFlag, cli.Value.KeepLast())
			clusterIPFlag.Description("ClusterIP to be assigned to the service. Leave empty to auto-allocate, or set to 'None' to create a headless service.")
			clusterIPFlag.Long("cluster-ip")

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRun, util.DryRunNone)
			dryRunFlag.Long("dry-run").ImplicitValue(util.DryRunClient)

			externalIPFlag := cb.Flags.String(&cb.Config.ExternalIP)
			cb.Values.ResolveWith(externalIPFlag, cli.Value.KeepLast())
			externalIPFlag.Description("Additional external IP address (not managed by Kubernetes) to accept for the service. If this IP is routed to a node, the service can be accessed by this IP in addition to its generated service IP.")
			externalIPFlag.Long("external-ip")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-expose")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("Filename, directory, or URL to files identifying the resource to expose a service")
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			labels := cb.Values.Value(&cb.Config.Labels, exposeLabelSetLanguage)
			labelsFlag := cb.Flags.ValueFor(labels)
			cb.Values.ResolveWith(labelsFlag, cli.Value.KeepLast())
			labelsFlag.Description("Labels to apply to the service created by this call.")
			labelsFlag.Long("labels")
			labelsFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			loadBalancerIPFlag := cb.Flags.String(&cb.Config.LoadBalancerIP)
			cb.Values.ResolveWith(loadBalancerIPFlag, cli.Value.KeepLast())
			loadBalancerIPFlag.Description("IP to assign to the LoadBalancer. If empty, an ephemeral IP will be created and used (cloud-provider specific).")
			loadBalancerIPFlag.Long("load-balancer-ip")

			nameFlag := cb.Flags.String(&cb.Config.Name)
			cb.Values.ResolveWith(nameFlag, cli.Value.KeepLast())
			nameFlag.Description("The name for the newly created object.")
			nameFlag.Long("name")

			overrideTypeFlag := cb.Flags.String(&cb.Config.OverrideTypeInput)
			cb.Values.ResolveWith(overrideTypeFlag, cli.Value.KeepLast())
			overrideTypeFlag.Description("The method used to override the generated object: json, merge, or strategic.")
			cb.Values.Default(overrideTypeFlag, string(util.DefaultOverrideType))
			overrideTypeFlag.Long("override-type")

			overridesFlag := cb.Flags.String(&cb.Config.Overrides)
			cb.Values.ResolveWith(overridesFlag, cli.Value.KeepLast())
			overridesFlag.Description("An inline JSON override for the generated object. If this is non-empty, it is used to override the generated object. Requires that the object supply a valid apiVersion field.")
			overridesFlag.Long("overrides")
			cb.Values.Derive(
				&cb.Config.OverrideType,
				cli.Value.ComputedBy(func(_ context.Context, cfg *exposeConfig, _ cli.Dependencies) (util.OverrideType, error) {
					if cfg.Overrides == "" || cfg.OverrideTypeInput == "" {
						return util.DefaultOverrideType, nil
					}
					return util.OverrideType(cfg.OverrideTypeInput), nil
				}).From(overrideTypeFlag, overridesFlag.OptionalDependency()),
			)

			port := cb.Values.Value(&cb.Config.Port, exposePortSetLanguage)
			portFlag := cb.Flags.ValueFor(port)
			cb.Values.ResolveWith(portFlag, cli.Value.KeepLast())
			portFlag.Description("The port that the service should serve on. Copied from the resource being exposed, if unspecified")
			portFlag.Long("port")

			protocolFlag := cb.Flags.String(&cb.Config.Protocol)
			cb.Values.ResolveWith(protocolFlag, cli.Value.KeepLast())
			protocolFlag.Description("The network protocol for the service to be created. Default is 'TCP'.")
			protocolFlag.Long("protocol")

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			cb.Values.ResolveWith(recordFlag, cli.Value.KeepLast())
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			selector := cb.Values.Value(&cb.Config.Selector, exposeLabelSetLanguage)
			selectorFlag := cb.Flags.ValueFor(selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("A label selector to use for this service. Only equality-based selector requirements are supported. If empty (the default) infer the selector from the resource being exposed.")
			selectorFlag.Long("selector")

			sessionAffinity := cb.Values.Value(&cb.Config.SessionAffinity, exposeSessionAffinityLanguage)
			sessionAffinityFlag := cb.Flags.ValueFor(sessionAffinity)
			cb.Values.ResolveWith(sessionAffinityFlag, cli.Value.KeepLast())
			sessionAffinityFlag.Description("If non-empty, set the session affinity for the service to this; legal values: 'None', 'ClientIP'")
			sessionAffinityFlag.Long("session-affinity")

			targetPortFlag := cb.Flags.String(&cb.Config.TargetPort)
			cb.Values.ResolveWith(targetPortFlag, cli.Value.KeepLast())
			targetPortFlag.Description("Name or number for the port on the container that the service should direct traffic to. Optional.")
			targetPortFlag.Long("target-port")

			typeFlag := cb.Flags.String(&cb.Config.Type)
			cb.Values.ResolveWith(typeFlag, cli.Value.KeepLast())
			typeFlag.Description("Type for this service: ClusterIP, NodePort, LoadBalancer, or ExternalName. Default is 'ClusterIP'.")
			typeFlag.Long("type")

			cb.Rule.When(cb.When.NotEqual(overridesFlag.Destination(), "")).In(
				overrideTypeFlag.Destination(),
				"",
				string(util.OverrideTypeJSON),
				string(util.OverrideTypeMerge),
				string(util.OverrideTypeStrategic),
			)
			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-expose-source", func(_ context.Context, cfg *exposeConfig, _ cli.Dependencies) (bool, error) {
				hasFilename := len(cfg.Filename.Filenames()) > 0
				hasKustomize := cfg.Filename.Kustomize != ""
				if hasFilename && hasKustomize {
					return false, nil
				}
				return len(cfg.Arguments) > 0 || hasFilename || hasKustomize, nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
		},
		Run: runExpose,
	}
	return cmd
}
