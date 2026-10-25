# Commandment findings exposed by kubectl pressure

Workstream 1 found two implementation defects and one style-authority inconsistency. It did not find a required new public API or semantic model.

## C1 — alternative positional completion retained a semantically rejected prior branch

Classification: `COMMANDMENT_IMPLEMENTATION`.

Kubectl `get` legitimately has both:

```text
TYPE NAME...
TYPE/NAME...
```

For completion of `get pods pod`, the qualified `TYPE/NAME...` form remained structurally viable even though its complete prior token `pods` is rejected by that form's value language. That speculative branch interfered with settlement of the ordinary form's `TYPE` dependency, preventing Kubernetes-backed name completion.

Rejected workarounds:

- a kubectl-specific completion parser: creates a second grammar/value model;
- weakening the kubectl operand grammar: distorts the migrated interface;
- a global completion Factory/registry: recreates the architecture Commandment is intended to remove;
- ignoring the required dependency: loses the selected resource type.

Bounded correction candidate:

```text
parent  a02adf48fc503a57a0898caf235f800b08205270
commit  18f17c04f553ce6d56e4487805f2f7f7c1515efc
```

The correction carries the complete prior argument allocation into completion analysis and prunes only alternatives whose already-complete allocation is semantically rejected before callbacks/dependencies are evaluated. It does not infer from the incomplete cursor token.

Returned proof:

```text
completion/core                 PASS
completion/dynamic              PASS
completion/dynamic/providers    PASS
production cli lint             0 issues
git diff --check                PASS
root external cli_test          NOT RUN (target-only sandbox law)
```

## C2 — failed optional producer dependency could panic

Classification: `COMMANDMENT_IMPLEMENTATION`.

Active DEC-B1 dependency law says `OptionalDependency` permits resolved absence or inactivity and does not swallow failure or blocking. A disposable kubectl mutation using a derived value exposed that a failed optional dependency could proceed toward callback construction and panic with:

```text
cli: value producer dependencies are not ready
```

A standalone non-kubectl conformance reproduction confirmed the defect.

Bounded correction candidate:

```text
parent  a02adf48fc503a57a0898caf235f800b08205270
commit  2d6984dff370a96cc23211bfab1a7ea6a5c805b2
```

The correction treats only `destinationStateAbsent` and `destinationStateInactive` as optional-tolerated unresolved states. Missing, failed, blocked, invalid, or otherwise unresolved dependency state blocks the dependent producer rather than reaching an impossible callback view.

Returned proof:

```text
runtime/values conformance      PASS
production cli lint             0 issues
git diff --check                PASS
root external cli_test          NOT RUN (target-only sandbox law)
```

## C3 — large-application style authority is internally inconsistent

Classification: `COMMANDMENT_STYLE_AUTHORITY`.

Current Go style and `docs/12-building-large-applications.md` both say static topology should remain lexical and helpers must earn semantic/operation/integration ownership. Active DEC-B1 §12 still prescribes thin orchestration helpers and illustrates:

```text
declareRootInputs
declareRootRules
declareRootCommands
```

The kubectl source and mutation campaign falsify that categorical orchestration pattern as the canonical default. `declareRoot` itself survives because root application composition is a real boundary; categorical child helpers do not.

No workspace/Commandment authority file is silently changed by this standalone POC. The required downstream authority action is a bounded DEC-B1 §12 correction so it agrees with the already-stronger current Go style and public large-application guide.

## Combined correction donor

The two implementation corrections compose without conflict on the exact Commandment baseline:

```text
parent  a02adf48fc503a57a0898caf235f800b08205270
C1      37f1abf164ce5a2b3bf371f46536d00d2277bc18
C2      73621550f7ceba1b58c9c71d92a669580beec981
tree    595d71d81bdff0788613e21bff715b90372d5bf0
```

Returned combined proof on Go 1.27.1 linux/amd64:

```text
completion/core                 PASS
completion/dynamic              PASS
completion/dynamic/providers    PASS
runtime/values                  PASS
production cli lint             0 issues
Linux cli build                 PASS
Windows/amd64 cli build         PASS
git diff --check                PASS
root external cli_test          NOT RUN (target-only sandbox law)
```

The kubectl source candidate is verified against this composed donor. Its committed `go.mod` intentionally remains bound to the authoritative pre-correction Commandment baseline until these changes are accepted/rebased through the normal Commandment integration path; broad kubectl migration should not proceed against the uncorrected baseline and should not hide either defect in application code.
