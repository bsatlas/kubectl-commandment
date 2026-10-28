package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type exposeConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	ClusterIP                string
	DryRun                   string
	ExternalIP               string
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Labels                   string
	LoadBalancerIP           string
	Name                     string
	Output                   string
	OverrideType             string
	Overrides                string
	Port                     string
	Protocol                 string
	Record                   bool
	Recursive                bool
	SaveConfig               bool
	Selector                 string
	SessionAffinity          string
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

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			clusterIPFlag := cb.Flags.String(&cb.Config.ClusterIP)
			clusterIPFlag = clusterIPFlag.Description("ClusterIP to be assigned to the service. Leave empty to auto-allocate, or set to 'None' to create a headless service.")
			clusterIPFlag.Long("cluster-ip")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			externalIPFlag := cb.Flags.String(&cb.Config.ExternalIP)
			externalIPFlag = externalIPFlag.Description("Additional external IP address (not managed by Kubernetes) to accept for the service. If this IP is routed to a node, the service can be accessed by this IP in addition to its generated service IP.")
			externalIPFlag.Long("external-ip")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-expose")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files identifying the resource to expose a service")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			labelsFlag := cb.Flags.String(&cb.Config.Labels)
			labelsFlag = labelsFlag.Description("Labels to apply to the service created by this call.")
			labelsFlag.Long("labels")
			labelsFlag.Short('l')

			loadBalancerIPFlag := cb.Flags.String(&cb.Config.LoadBalancerIP)
			loadBalancerIPFlag = loadBalancerIPFlag.Description("IP to assign to the LoadBalancer. If empty, an ephemeral IP will be created and used (cloud-provider specific).")
			loadBalancerIPFlag.Long("load-balancer-ip")

			nameFlag := cb.Flags.String(&cb.Config.Name)
			nameFlag = nameFlag.Description("The name for the newly created object.")
			nameFlag.Long("name")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			overrideTypeFlag := cb.Flags.String(&cb.Config.OverrideType)
			overrideTypeFlag = overrideTypeFlag.Description("The method used to override the generated object: json, merge, or strategic.")
			cb.Values.Default(overrideTypeFlag, "merge")
			overrideTypeFlag.Long("override-type")

			overridesFlag := cb.Flags.String(&cb.Config.Overrides)
			overridesFlag = overridesFlag.Description("An inline JSON override for the generated object. If this is non-empty, it is used to override the generated object. Requires that the object supply a valid apiVersion field.")
			overridesFlag.Long("overrides")

			portFlag := cb.Flags.String(&cb.Config.Port)
			portFlag = portFlag.Description("The port that the service should serve on. Copied from the resource being exposed, if unspecified")
			portFlag.Long("port")

			protocolFlag := cb.Flags.String(&cb.Config.Protocol)
			protocolFlag = protocolFlag.Description("The network protocol for the service to be created. Default is 'TCP'.")
			protocolFlag.Long("protocol")

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag = recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag = recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("A label selector to use for this service. Only equality-based selector requirements are supported. If empty (the default) infer the selector from the resource being exposed.")
			selectorFlag.Long("selector")

			sessionAffinityFlag := cb.Flags.String(&cb.Config.SessionAffinity)
			sessionAffinityFlag = sessionAffinityFlag.Description("If non-empty, set the session affinity for the service to this; legal values: 'None', 'ClientIP'")
			sessionAffinityFlag.Long("session-affinity")

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			targetPortFlag := cb.Flags.String(&cb.Config.TargetPort)
			targetPortFlag = targetPortFlag.Description("Name or number for the port on the container that the service should direct traffic to. Optional.")
			targetPortFlag.Long("target-port")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			typeFlag := cb.Flags.String(&cb.Config.Type)
			typeFlag = typeFlag.Description("Type for this service: ClusterIP, NodePort, LoadBalancer, or ExternalName. Default is 'ClusterIP'.")
			typeFlag.Long("type")
		},
		Run: runExpose,
	}
	return cmd
}
