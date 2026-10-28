package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type runConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	Arguments                []string
	AllowMissingTemplateKeys bool
	Annotations              []string
	Attach                   bool
	AttachState              cli.OptionalValue[cli.ValueState]
	Cascade                  string
	Command                  bool
	DetachKeys               string
	DryRun                   string
	Env                      []string
	Expose                   bool
	FieldManager             string
	Filename                 []string
	Force                    bool
	GracePeriod              int
	Image                    string
	ImagePullPolicy          string
	Kustomize                string
	Labels                   string
	LeaveStdinOpen           bool
	Output                   string
	OverrideType             string
	Overrides                string
	PodRunningTimeout        time.Duration
	Port                     string
	Privileged               bool
	Quiet                    bool
	Record                   bool
	Recursive                bool
	Restart                  string
	Rm                       bool
	SaveConfig               bool
	ShowManagedFields        bool
	Stdin                    bool
	Template                 string
	Timeout                  time.Duration
	TTY                      bool
	Wait                     bool
}

func runCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[runConfig] {
	cmd := cli.Command[runConfig]{
		Name: "run",
		Description: cli.Description{
			Short: "Run a particular image on the cluster",
			Long:  "Create and run a particular image in a pod.",
		},
		Examples: []cli.Example{
			{
				Description: "Start a nginx pod",
				CommandLine: "kubectl run nginx --image=nginx",
			},
			{
				Description: "Start a hazelcast pod and let the container expose port 5701",
				CommandLine: "kubectl run hazelcast --image=hazelcast/hazelcast --port=5701",
			},
			{
				Description: "Start a hazelcast pod and set environment variables \"DNS_DOMAIN=cluster\" and \"POD_NAMESPACE=default\" in the container",
				CommandLine: "kubectl run hazelcast --image=hazelcast/hazelcast --env=\"DNS_DOMAIN=cluster\" --env=\"POD_NAMESPACE=default\"",
			},
			{
				Description: "Start a hazelcast pod and set labels \"app=hazelcast\" and \"env=prod\" in the container",
				CommandLine: "kubectl run hazelcast --image=hazelcast/hazelcast --labels=\"app=hazelcast,env=prod\"",
			},
			{
				Description: "Dry run; print the corresponding API objects without creating them",
				CommandLine: "kubectl run nginx --image=nginx --dry-run=client",
			},
			{
				Description: "Start a nginx pod, but overload the spec with a partial set of values parsed from JSON",
				CommandLine: "kubectl run nginx --image=nginx --overrides='{ \"apiVersion\": \"v1\", \"spec\": { ... } }'",
			},
			{
				Description: "Start a busybox pod and keep it in the foreground, don't restart it if it exits",
				CommandLine: "kubectl run -i -t busybox --image=busybox --restart=Never",
			},
			{
				Description: "Start the nginx pod using the default command, but use custom arguments (arg1 .. argN) for that command",
				CommandLine: "kubectl run nginx --image=nginx -- <arg1> <arg2> ... <argN>",
			},
			{
				Description: "Start the nginx pod using a different command and custom arguments",
				CommandLine: "kubectl run nginx --image=nginx --command -- <cmd> <arg1> ... <argN>",
			},
		},
		Declare: func(cb *cli.CommandBuilder[runConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[runConfig]) {
				form.String(&form.Config.Name, "NAME")
				form.PassThrough(arguments, "ARG", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			annotationsValue := cb.Values.Sequence(&cb.Config.Annotations, cli.Value.String())
			annotationsFlag := cb.Flags.Sequence(annotationsValue)
			annotationsFlag = annotationsFlag.Description("Annotations to apply to the pod.")
			annotationsFlag.Long("annotations")

			attachFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Attach))
			attachFlag = attachFlag.Description("If true, wait for the Pod to start running, and then attach to the Pod as if 'kubectl attach ...' were called.  Default false, unless '-i/--stdin' is set, in which case the default is true. With '--restart=Never' the exit code of the container process is returned.")
			attachFlag.Long("attach").ImplicitValue(true)
			cb.Access.ValueState(attachFlag, &cb.Config.AttachState)

			cascadeFlag := cb.Flags.String(&cb.Config.Cascade)
			cascadeFlag = cascadeFlag.Description("Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents (e.g. Pods created by a ReplicationController). Defaults to background.")
			cb.Values.Default(cascadeFlag, "background")
			cascadeFlagLong := cascadeFlag.Long("cascade")
			cascadeFlagLong.ImplicitValue("background")

			commandFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Command))
			commandFlag = commandFlag.Description("If true and extra arguments are present, use them as the 'command' field in the container, rather than the 'args' field which is the default.")
			commandFlag.Long("command").ImplicitValue(true)

			detachKeysFlag := cb.Flags.String(&cb.Config.DetachKeys)
			detachKeysFlag = detachKeysFlag.Description("Override the key sequence for detaching a container.")
			cb.Values.Default(detachKeysFlag, "ctrl-p,ctrl-q")
			detachKeysFlag.Long("detach-keys")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			envValue := cb.Values.Sequence(&cb.Config.Env, cli.Value.String())
			envFlag := cb.Flags.Sequence(envValue)
			envFlag = envFlag.Description("Environment variables to set in the container.")
			envFlag.Long("env")

			exposeFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Expose))
			exposeFlag = exposeFlag.Description("If true, create a ClusterIP service associated with the pod.  Requires `--port`.")
			exposeFlag.Long("expose").ImplicitValue(true)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-run")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("to use to replace the resource.")
			filenameFlag = filenameFlag.Hidden().Release(preexistingDeprecation("it is ignored by kubectl run and will be removed in a future release"))
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			forceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Force))
			forceFlag = forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)

			gracePeriodFlag := cb.Flags.Int(&cb.Config.GracePeriod)
			gracePeriodFlag = gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			cb.Values.Default(gracePeriodFlag, -1)
			gracePeriodFlag.Long("grace-period")

			imageFlag := cb.Flags.String(&cb.Config.Image)
			imageFlag = imageFlag.Description("The image for the container to run.")
			imageFlag.Long("image")

			imagePullPolicyFlag := cb.Flags.String(&cb.Config.ImagePullPolicy)
			imagePullPolicyFlag = imagePullPolicyFlag.Description("The image pull policy for the container.  If left empty, this value will not be specified by the client and defaulted by the server.")
			imagePullPolicyFlag.Long("image-pull-policy")

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			labelsFlag := cb.Flags.String(&cb.Config.Labels)
			labelsFlag = labelsFlag.Description("Comma separated labels to apply to the pod. Will override previous values.")
			labelsFlag.Long("labels")
			labelsFlag.Short('l')

			leaveStdinOpenFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.LeaveStdinOpen))
			leaveStdinOpenFlag = leaveStdinOpenFlag.Description("If the pod is started in interactive mode or with stdin, leave stdin open after the first attach completes. By default, stdin will be closed after the first attach completes.")
			leaveStdinOpenFlag.Long("leave-stdin-open").ImplicitValue(true)

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

			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			podRunningTimeoutFlag = podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, time.Minute)
			podRunningTimeoutFlag.Long("pod-running-timeout")

			portFlag := cb.Flags.String(&cb.Config.Port)
			portFlag = portFlag.Description("The port that this container exposes.")
			portFlag.Long("port")

			privilegedFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Privileged))
			privilegedFlag = privilegedFlag.Description("If true, run the container in privileged mode.")
			privilegedFlag.Long("privileged").ImplicitValue(true)

			quietFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Quiet))
			quietFlag = quietFlag.Description("If true, suppress prompt messages.")
			quietFlag.Long("quiet").ImplicitValue(true)
			quietFlag.Short('q').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag = recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag = recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			restartFlag := cb.Flags.String(&cb.Config.Restart)
			restartFlag = restartFlag.Description("The restart policy for this Pod.  Legal values [Always, OnFailure, Never].")
			cb.Values.Default(restartFlag, "Always")
			restartFlag.Long("restart")

			rmFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Rm))
			rmFlag = rmFlag.Description("If true, delete the pod after it exits.  Only valid when attaching to the container, e.g. with '--attach' or with '-i/--stdin'.")
			rmFlag.Long("rm").ImplicitValue(true)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			stdinFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Stdin))
			stdinFlag = stdinFlag.Description("Keep stdin open on the container in the pod, even if nothing is attached.")
			stdinFlag.Long("stdin").ImplicitValue(true)
			stdinFlag.Short('i').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag = timeoutFlag.Description("The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
			timeoutFlag.Long("timeout")

			ttyFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.TTY))
			ttyFlag = ttyFlag.Description("Allocate a TTY for the container in the pod.")
			ttyFlag.Long("tty").ImplicitValue(true)
			ttyFlag.Short('t').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			waitFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Wait))
			waitFlag = waitFlag.Description("If true, wait for resources to be gone before returning. This waits for finalizers.")
			waitFlag.Long("wait").ImplicitValue(true)
		},
		Run: runRun,
	}
	return cmd
}
