# Workstream 1 deliberate mutation campaign

The campaign tests edit locality and conceptual locality. Disposable mutations were performed from the implementation checkpoint and then reverted unless they had a real kubectl owner. Verification-only `go.mod` version shims used to consume the supplied Kubernetes vendor tree are excluded from the source-diff observations below.

| Pressure | Result | Files/owners affected | Final disposition |
| --- | --- | --- | --- |
| Add real config value/command pressure | Disposable `config delete-context NAME` used the existing config service and one lexical argument declaration; focused config/cmd build passed | `internal/config/command.go` only | REVERT; locality confirmed, no new abstraction earned |
| Add config sibling | Real `config current-context` added after initial root/get composition; tests passed | config package + tests; root staging unchanged because config parent owns siblings | KEEP; real kubectl behavior and strong topology evidence |
| Add `get` policy relationship | Disposable `show-kind`/output relation added as one declaration rule; full tests passed | `internal/get/command.go` only | REVERT artificial relation; policy changes stay local |
| Make later declaration depend on prior subject | Disposable derived human-readable value tested output/template dependencies | get declaration only | REVERT; exposed independent Commandment optional-dependency implementation defect rather than application-architecture need |
| Add root/global semantic value consumed by child | Disposable root request-timeout destination was passed explicitly into `get`; resource type/name completion remained green | root config, root assembly, get config/declaration | REVERT artificial pressure; explicit root-to-child dependency remained narrow, no global bag |
| Change ordinary application service | Disposable config loader service was inserted behind `Services` | `internal/config` only | REVERT; ordinary Go dependency change stayed inside application owner |
| Extract candidate declaration helper | Resource argument declaration was extracted into a helper returning a carrier | `internal/get/command.go` | INLINE/REJECT; helper hid grammar and created staging-only state |
| Inline existing helper | `printFlags` construction was inlined into one caller | `internal/get/runtime.go` | KEEP helper; the name owns reusable Kubernetes printer configuration and inlining obscured behavior |
| Move command/subtree boundary | `current-context` moved to a new package | config owner + new package | REVERT; split duplicated the cohesive kubeconfig service boundary and made ownership worse |
| Add semantic expectation | `semantic_test.go` proves compiled application command/grammar/flag facts with `clitest` | test only | KEEP; no parallel command registry introduced |
| Exercise completion after dependency change | Root request-timeout mutation was followed by Kubernetes-backed resource type/name completion | root/get only | PASS; completion remained on compiled semantic model |
| Introduce provider-produced static command | Disposable root provider returned one static probe command | `app.go` only | REVERT; technically works but providers are not source organization for static topology |

## Hostile counterfactual: remove `ClientSelection`

A further subtraction attack replaced the local derived `ClientSelection` destination with three execution `Access.Value` fields (`kubeconfig`, `context`, `namespace`). Source became smaller, but Kubernetes-backed resource type and resource-name completion failed because execution access is not the analysis-time value transport for completion callbacks.

That failure proves the aggregate is not a generic declaration-state bag. It is a real invocation semantic subject: the finalized Kubernetes client selection used by both completion and execution. Keeping one local derived `ClientSelection` preserves explicit dependency ownership without a global Factory or completion-only registry.

## Mutation conclusion

Common edits remained local to their semantic owners. The campaign did not require a generic declaration context, service locator, global mutable state, callback-capability retention, static providers for modularity, or a helper mini-DSL. Where extraction was tried solely for source compression, hostile comparison favored the lexical form.
