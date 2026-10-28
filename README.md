# kubectl-commandment

`kubectl-commandment` is a standalone reference migration of the kubectl frontend from Cobra/pflag to [Commandment](https://github.com/abstractionsinc/commandment).

The repository keeps Commandment as the single authoritative CLI semantic model while reusing Kubernetes application and domain behavior wherever that behavior is frontend-neutral. The executable path does not construct a shadow Cobra command tree, does not parse with pflag, and does not maintain a second command registry for Help, completion, or tests.

## Bound source revisions

The current frontend is bound to:

```text
Kubernetes   3930cfbbb185e71ecb7b993a5582de6f775bc1f0
Commandment  2db386083a0954fb010caa6894145ea6944b3999
Go           1.27.1
```

Current Git remains authoritative when the repository is rebased.

## Frontend coverage

The bound Kubernetes tree contains 111 captured kubectl command nodes. This repository represents that topology as:

```text
1   root Commandment Definition
109 explicit static cli.Command declarations
1   intrinsic Commandment shell-completion command
---
111 captured command nodes represented
```

Static parent commands such as `config`, `create`, `rollout`, `set`, and `top` remain structural namespaces when Kubernetes does not assign them independent business execution. `options` likewise remains structural and uses the Commandment Help projection to render the inherited root options. User-installed kubectl plugins are materialized dynamically through Commandment command providers while Kubernetes continues to own plugin discovery and process execution semantics.

The migrated frontend includes the root/global kubectl configuration surface and the built-in command families represented by the bound Kubernetes tree, including configuration, creation, read/discovery, mutation, rollout, streaming/process, networking, apply/diff/edit, debug, and plugin behavior.

## Source architecture

The frontend intentionally uses ordinary Go source rather than mirroring the command tree as packages:

- `cmd/kubectl/main.go` owns process adaptation only;
- `cmd/kubectl/cli.go` owns the root definition, global semantic values, and visible top-level staging;
- parent command files visibly stage their static children;
- command declarations keep Commandment grammar, values, policy, completion, and access relationships visible;
- substantial Kubernetes execution behavior lives in semantically named `*_run.go` or operation files in the same package;
- command configuration owns admitted CLI semantics, while Kubernetes clients and services remain ordinary Go dependencies;
- functions are used where they bind real lifetime state, such as callback-scoped Commandment destinations; static definitions otherwise remain values;
- dynamic providers are reserved for genuinely dynamic plugin namespaces, not static source organization.

Where upstream kubectl option structs expose frontend-neutral runtime methods, the migration calls those methods directly. Where upstream preparation is coupled to Cobra or private parser state, the frontend performs only the required Commandment-owned preparation and then returns to Kubernetes resource/client/printer/domain code. Cobra is not used as an execution adapter.

## Compatibility verification

[`compatibility/compare_test.go`](compatibility/compare_test.go) is a subprocess differential oracle for selected compatibility-sensitive behavior. It compares a legacy kubectl binary and this candidate against equivalent deterministic fixtures when `KUBECTL_LEGACY` and `KUBECTL_COMMANDMENT` are provided.

That suite is intentionally a selected compatibility oracle, not a claim that every kubectl presentation byte or every target-platform process edge has been differentially frozen. The repository also relies on Commandment semantic tests and focused family verification at the smallest truthful ownership boundary. See [`compatibility/README.md`](compatibility/README.md) for the current oracle contract.

The bound Commandment revision includes the generic framework corrections originally exposed by this migration for kubectl-compatible implicit-value flag spellings, semantic dependency readiness, and `--` transfer semantics. The frontend retains no application-side parser or dependency workaround; the focused flag and separator regressions remain in this repository as compatibility proof against the accepted framework behavior.

## Verification posture

The migrated source has been partitioned into focused command-family compile slices against the exact Kubernetes vendor graph and current Commandment source. Every file owning an explicit `cli.Command` declaration is included in at least one green focused slice.

A constrained sandbox is not a truthful whole-package performance or acceptance environment for this dependency graph. Final release/adoption acceptance therefore still requires the normal full repository test/build/lint gates in a target environment where they can complete, plus any platform/shell/performance evidence required for an upstream adoption claim.

This repository demonstrates the full Commandment frontend source shape for the bound kubectl tree. It is not an upstream Kubernetes fork and, by itself, does not claim SIG CLI production-adoption readiness or resolve the separate ecosystem transition for exported Kubernetes APIs that expose Cobra/pflag types.
