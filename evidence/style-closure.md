# Kubectl large-application style closure

## Surviving source grammar

1. `main` adapts process concerns and runs one compiled application. It does not own the command tree.
2. One `declareRoot` boundary is earned because application composition and genuinely root-scoped semantics are one recognizable owner.
3. Root keeps its real declarations and top-level static topology visible. It does not delegate to categorical `declareRootInputs`, `declareRootRules`, or `declareRootCommands` layers.
4. A static command package exposes the smallest ordinary Go construction boundary required to bind services/root semantic dependencies; its `Command.Declare` callback remains the front door for actual Commandment grammar, values, policy, completion, and access declarations.
5. Command configuration owns the admitted semantic inputs its operation needs. Application clients/services remain ordinary Go dependencies.
6. Ancestor semantic dependencies are passed explicitly. A local derived aggregate is justified only when it is itself a real semantic subject needed across Commandment operations; `get.ClientSelection` survives this test because completion and execution both consume the finalized client selection.
7. Helpers earn extraction by naming a real operation, reusable construction, algorithm, integration boundary, invariant, or lifetime owner. Category helpers and staging-only carriers fail the mental-inline test.
8. Static topology stays static. Providers are reserved for invocation-dependent/external namespaces.
9. Completion remains attached to the command semantic declaration and receives only declared dependencies; no broad package-global Factory or second parser/value model exists.
10. Tests/tooling query the compiled application (`clitest`, `Complete`) instead of maintaining a parallel command registry.

## Exact source evidence

### Root

`app.go` visibly owns:

- interspersed flag-order dialect;
- root `--kubeconfig`, `--context`, and `--namespace`/`-n` semantics;
- static staging of `config` and `get`;
- explicit root destinations supplied to child construction.

### Simple control: `config use-context`

The command's `Declare` callback directly shows:

- root kubeconfig access;
- stdout access;
- a kubeconfig-backed dynamic value domain;
- the context positional operand.

The same dynamic semantic relation provides validation and completion. Mutation and persistence use Kubernetes `clientcmd` code outside declaration machinery.

### Stress subject: `get`

The `Declare` callback keeps the complete researched local `NewCmdGet` input surface visible, including operand alternatives, all selected local flags, defaults, completion dependencies, and validation relationships. Ordinary resource building, REST, printer configuration, sorting, raw HTTP, and rendering live in ordinary Go runtime/service functions.

The declaration is long, but the mutation campaign found no categorical extraction that made it easier to understand. The direct form is preferable to a helper table of contents.

## Hostile attacks and dispositions

| Attack | Result |
| --- | --- |
| Inline one-use declaration helpers | improves or preserves clarity; weak helpers removed/rejected |
| Extract resource grammar helper | worse; hides grammar and creates staging-only carrier |
| Inline `printFlags` | worse; hides a real reusable Kubernetes printer-construction operation |
| Remove `ClientSelection` derived semantic aggregate | worse/incorrect; breaks analysis-time completion dependency delivery |
| Move `current-context` into separate package | worse; splits cohesive config ownership and duplicates service boundary |
| Replace static child with provider | worse; dynamic mechanism used only for source organization |
| Add root semantic dependency | stays explicit/local; no universal root/child state bag required |
| Change ordinary service | stays inside ordinary Go owner |

## Authority disposition

| Owner | Disposition | Reason |
| --- | --- | --- |
| `spec/GO_STYLE_GUIDE.md` | ALREADY CONSISTENT | current source-form law explicitly allows long orchestration when visible ownership/order is clearer and requires extraction to name a real semantic/operation/lifetime boundary |
| `commandment/docs/12-building-large-applications.md` | ALREADY CONSISTENT | current guide rejects both giant-root ownership and forests of one-use helpers, prefers lexical static topology, and keeps services ordinary Go |
| active `DEC-B1-001` §12 | CHANGE | categorical “thin orchestration + typed helper” prescription and `declareRootInputs`/`declareRootRules`/`declareRootCommands` example are broader than the style that survived real kubectl pressure |
| focused examples | ALREADY CONSISTENT | examples remain lexical capability demonstrations and should not be replaced by kubectl as the basic learning corpus |

The needed DEC-B1 correction is narrow: make lexical declaration the default, retain synchronous helpers when a real boundary earns them, and treat `declareRoot`-level orchestration as an application-composition choice rather than a mandate to split declarations by framework category.

## Verdict

```text
KUBECTL LARGE-APPLICATION STYLE CLOSURE: YES
FULL-CLONE CONTINUATION: YES
SIG-CLI ADOPTION READINESS: BLOCKED
```

No unresolved architecture-bearing question remains in Workstream 1. Two bounded Commandment implementation defects are separately identified with a composed correction donor; neither requires a kubectl-specific workaround or a new public authoring model. Full-clone continuation should begin only after that donor is accepted/rebased into current Commandment and the kubectl repository is rebound to the accepted revision.

SIG CLI adoption readiness remains blocked by deliberately unfinished evidence: complete built-in command/root behavior, `.kuberc` and plugin compatibility, exhaustive shell/process/platform compatibility, the upstream Cobra/pflag API-transition ledger, and decision-grade kubectl-shape performance/build evidence. These are adoption blockers, not large-application source-style blockers.

The next migration family is **Workstream A — finish the config/context family**. `current-context` is already present from the mutation campaign; after recapturing current Kubernetes Git, the next ordinary sibling pressure should begin with `get-contexts` if that command remains current, then continue through the coherent config subtree before opening unrelated command families.
