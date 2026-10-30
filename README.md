# kubectl-commandment

`kubectl-commandment` is a reference migration of kubectl's CLI frontend from its existing Cobra/pflag implementation to [Commandment](https://github.com/abstractionsinc/commandment).

The migration keeps Kubernetes runtime and domain behavior where it already has the right owner. It changes where kubectl's frontend semantics live: grammar, values, resolution, relationships, completion, Help, and admission are represented by one compiled Commandment application before execution begins.

The bound Kubernetes tree contains 111 captured built-in/intrinsic kubectl command nodes. This repository represents that captured topology without constructing a shadow Cobra/pflag command tree. The production `cmd/kubectl` frontend has no Cobra or pflag imports.

## Same kubectl frontend, different ownership model

A common kubectl command shape built on Cobra/pflag is:

```text
pflag mutation + Args + completion callbacks
                    |
                    v
                 Options
                    |
                    v
             Complete / Validate
                    |
                    v
             Kubernetes operation
```

This repository targets:

```text
Commandment declaration
        |
        v
resolve / derive / validate
        |
        v
admitted typed configuration
        |
        v
Kubernetes operation
```

Same bound kubectl frontend contract. Different ownership model.

Facts knowable from the invocation are admitted before `Run`. Filesystem, discovery, API, authorization, terminal, stream, and other runtime facts remain runtime concerns.

| kubectl concern | Cobra/pflag kubectl | This migration |
| --- | --- | --- |
| Argument grammar | command metadata, validators, and application logic | executable argument forms |
| Repeated values | native pflag behavior | explicit resolution policy |
| Completion | command/flag completion callbacks | completion from semantic owners and dependencies |
| Frontend validation | may span `Args`, `Complete`, `Validate`, and runtime assumptions | admission before `Run` |
| Runtime/domain work | Kubernetes operations | Kubernetes operations |

Neither model makes the other incapable of implementing kubectl. The experiment compares where the same frontend meaning lives.

## Compare the implementations

### `kubectl config set-context`

`config set-context` is a compact example because several frontend concerns meet in one command: an optional positional context name, `--current`, completion from kubeconfig state, repeated-value behavior, and the relationship between the positional name and the settled Boolean value.

The Commandment implementation gives those facts one frontend owner:

```go
names := cb.Values.Sequence(&cb.Config.Names, cli.Value.String())
cb.Arguments.Sequence(names, "NAME", cli.Count.Optional()).
    CompleteBy(kubeconfigNameCompleter(
        func(cfg *configSetContextConfig) string { return cfg.Kubeconfig },
        kubeconfigContextName,
    )).
    From(kubeconfig)

currentValue := cb.Values.Bool(&cb.Config.Current)
current := cb.Flags.ValueFor(currentValue)
cb.Values.ResolveWith(current, cli.Value.KeepLast())
current.Long("current").ImplicitValue(true)

cb.Rule.Check("kubectl-config-set-context-name", func(_ context.Context, cfg *configSetContextConfig, _ cli.Dependencies) (bool, error) {
    name := ""
    if len(cfg.Names) == 1 {
        name = cfg.Names[0]
    }
    return (name == "") == cfg.Current, nil
}).
    DependsOn(names.OptionalDependency()).
    DependsOn(currentValue.OptionalDependency())
```

The command-local tests distinguish completion from validity and occurrence from settled meaning. A kubeconfig context may be suggested without becoming an acceptance constraint, and `--current=false` is not treated as if `--current` were true merely because the flag occurred.

- [Commandment implementation](cmd/kubectl/config_set_context.go)
- [Semantic propositions](cmd/kubectl/config_set_context_test.go)
- [Bound upstream Cobra implementation](https://github.com/kubernetes/kubernetes/blob/3930cfbbb185e71ecb7b993a5582de6f775bc1f0/staging/src/k8s.io/kubectl/pkg/cmd/config/set_context.go#L56-L161)

### `kubectl create deployment`

`create deployment` demonstrates a different boundary: changing frontend ownership does not require rewriting the Kubernetes operation.

The frontend owns the invocation semantics:

```text
NAME
NAME -- COMMAND...
--image
--replicas
--dry-run
input relationships
repeated-value policy
```

After admission, the execution path still hands settled configuration to Kubernetes' existing operation type:

```go
options := create.NewCreateDeploymentOptions(cfg.Streams)

options.Name = cfg.Name
options.Images = slices.Clone(cfg.Images)
options.Replicas = cfg.Replicas
options.Command = slices.Clone(cfg.Command)

return options.Run()
```

Commandment replaces the frontend ownership model; it does not turn Kubernetes operations into framework code.

- [Commandment implementation](cmd/kubectl/create_deployment.go)
- [Semantic propositions](cmd/kubectl/create_deployment_test.go)
- [Bound upstream Cobra implementation](https://github.com/kubernetes/kubernetes/blob/3930cfbbb185e71ecb7b993a5582de6f775bc1f0/staging/src/k8s.io/kubectl/pkg/cmd/create/create_deployment.go#L95-L209)

## What the migration found

| Finding | Result | Evidence |
| --- | --- | --- |
| Frontend semantic ownership | More grammar and policy can be decided before `Run` instead of remaining in late preparation or validation stages. | [`config_set_context.go`](cmd/kubectl/config_set_context.go), [`get.go`](cmd/kubectl/get.go) |
| Completion and validity are different concerns | Completion candidates can come from semantic dependencies without becoming acceptance constraints. | [`config_set_context.go`](cmd/kubectl/config_set_context.go), [`config_set_context_test.go`](cmd/kubectl/config_set_context_test.go) |
| pflag compatibility becomes explicit | Last-occurrence behavior, short value forms, CSV-style values, and other parser conventions must be represented deliberately. This is clearer, but it is more application authoring than native pflag. | [`config_set_context_test.go`](cmd/kubectl/config_set_context_test.go), [`filename_options.go`](cmd/kubectl/filename_options.go) |
| Kubernetes runtime code remains reusable | Existing Kubernetes operation types can consume admitted configuration without using Cobra as an execution adapter. | [`create_deployment.go`](cmd/kubectl/create_deployment.go), [`rollout_status_run.go`](cmd/kubectl/rollout_status_run.go) |
| Large commands do not require a second semantic model | Resource grammar, output policy, completion, and cross-value rules can stay with the command while substantial execution remains ordinary Go. | [`get.go`](cmd/kubectl/get.go), [`get_run.go`](cmd/kubectl/get_run.go), [`get_test.go`](cmd/kubectl/get_test.go) |
| Dynamic kubectl plugins remain dynamic | Plugin discovery feeds dynamic Commandment command providers while static commands remain ordinary source. | [`plugin_provider.go`](cmd/kubectl/plugin_provider.go), [`plugin_provider_test.go`](cmd/kubectl/plugin_provider_test.go) |

## Source layout

Command source names derive from their complete command path. A separate `*_run.go` exists only when execution earns its own reader journey.

```text
cmd/kubectl/
    kubectl.go

    config_set_context.go
    config_set_context_test.go

    create_deployment.go
    create_deployment_test.go

    get.go
    get_run.go
    get_test.go

    proxy.go
    proxy_run.go
    proxy_test.go
```

```text
foo.go       command/frontend owner
foo_test.go  durable command propositions
foo_run.go   substantial post-admission execution, when it earns the split
```

The frontend lives in `cmd/kubectl`. Command-path-derived filenames keep command owners predictable without creating a Go package per command namespace.

## Explore

| Source | Why look here |
| --- | --- |
| [`cmd/kubectl/kubectl.go`](cmd/kubectl/kubectl.go) | root application assembly and global kubectl configuration |
| [`cmd/kubectl/get.go`](cmd/kubectl/get.go) | a large static command with resource grammar, output semantics, completion, and policy in one owner |
| [`cmd/kubectl/proxy.go`](cmd/kubectl/proxy.go) and [`proxy_run.go`](cmd/kubectl/proxy_run.go) | a small frontend in front of a long-lived network operation |
| [`cmd/kubectl/plugin_provider.go`](cmd/kubectl/plugin_provider.go) | dynamic plugin topology using Commandment command providers |
| [`compatibility/compare_test.go`](compatibility/compare_test.go) | selected subprocess differential checks against a legacy kubectl binary |

For the Commandment programming model itself, see the [Commandment repository](https://github.com/abstractionsinc/commandment).

## Scope

The frontend is currently bound to:

```text
Kubernetes   3930cfbbb185e71ecb7b993a5582de6f775bc1f0
Commandment  2db386083a0954fb010caa6894145ea6944b3999
Go           1.27.1
```

This is a reference migration and design experiment, not an upstream kubectl replacement proposal.

[`compatibility/compare_test.go`](compatibility/compare_test.go) provides selected process-level differential checks where a legacy executable is the useful independent oracle. It is not a claim of byte-for-byte coverage of every kubectl behavior; see [`compatibility/README.md`](compatibility/README.md) for the current scope.

Some exact kubectl behaviors expose known limitations in the bound Commandment revision. The migration does not install a second application-side parser to hide them; those cases remain framework-owned until the correct semantic owner can represent them.
