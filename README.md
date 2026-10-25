# kubectl-commandment

`kubectl-commandment` is an executable migration study of the kubectl frontend from Cobra/pflag to [Commandment](https://github.com/abstractionsinc/commandment).

The repository is being built in two stages. Workstream 1 uses the real kubectl root, `config use-context`, and `get` as pressure tests to close a maintainable large-application Commandment source style. Broad command migration begins only after that style survives compatibility, mutation, and hostile source review.

The frontend treats Commandment as the single CLI semantic model. Kubernetes kubeconfig, discovery, resource-builder, printer, REST, and other application/domain behavior remains Kubernetes code wherever it is frontend-neutral. There is no shadow Cobra tree, no pflag parsing in the Commandment path, and no second command registry for completion or tests.

## Workstream 1 status

The large-application style workstream is closed on the exact source candidate:

```text
KUBECTL LARGE-APPLICATION STYLE CLOSURE: YES
FULL-CLONE CONTINUATION: YES
SIG-CLI ADOPTION READINESS: BLOCKED
```

The verified kubectl source commit is `aac6731c524e03f6039d79be1b52a2bb0d799700`. Verification composes it with the bounded Commandment correction donor `73621550f7ceba1b58c9c71d92a669580beec981`; the committed module requirement remains on the authoritative pre-correction Commandment revision until those corrections are accepted through Commandment's normal integration path.

The current slice includes:

- root `--kubeconfig`, `--context`, and `--namespace` / `-n` semantics with interspersed flag order;
- real `config use-context` including alias, kubeconfig-backed context validation/completion, persistence through `clientcmd.ModifyConfig`, and exact selected success output;
- real `config current-context`, retained from the mutation campaign as a second config sibling;
- the complete researched local `NewCmdGet` input/flag surface for the bound Kubernetes revision;
- Commandment-owned admission relationships for the selected `get` validation contract;
- real Kubernetes discovery/resource-name completion without a package-global Factory;
- representative `get` execution through Kubernetes REST/resource-builder/printer machinery;
- raw REST execution through Kubernetes `rawhttp`;
- a subprocess differential oracle against the exact Cobra kubectl baseline.

Exhaustive `get` watch/stream and presentation parity, complete root globals, `.kuberc`, plugins, ecosystem transition, full command coverage, and decision-grade scale/performance evidence remain later work by design.

## Large-application source result

The surviving source grammar is deliberately ordinary:

- `main` is process adaptation only;
- `declareRoot` is the application-composition boundary and keeps root semantics/topology visible;
- static parents visibly stage static children;
- child `Declare` callbacks show their actual Commandment grammar, values, policy, completion, and access relationships;
- helpers exist for real Kubernetes operations/construction, not framework categories such as “flags” or “rules”;
- command configuration owns admitted semantic inputs while Kubernetes clients/services remain ordinary Go;
- dynamic providers are not used to organize static source.

See [`evidence/style-closure.md`](evidence/style-closure.md) and [`evidence/mutation-campaign.md`](evidence/mutation-campaign.md) for the hostile source/mutation proof.

## Compatibility and Commandment feedback

The pre-implementation compatibility ledger is in [`compatibility/workstream1.md`](compatibility/workstream1.md). The executable differential oracle is [`compatibility/compare_test.go`](compatibility/compare_test.go), and the returned evidence is summarized in [`evidence/differential-results.md`](evidence/differential-results.md).

Real kubectl pressure exposed two bounded Commandment implementation defects rather than reasons to add application workarounds: alternative positional completion needed to prune semantically rejected prior allocations, and optional producer dependencies needed to preserve failure/blocking instead of reaching an internal panic. Both are isolated with focused correction candidates and evidence in [`evidence/commandment-findings.md`](evidence/commandment-findings.md).

This repository is research/reference source. It is not an upstream Kubernetes fork and does not claim upstream adoption readiness while the broad compatibility, performance, and ecosystem-transition programs remain open.
