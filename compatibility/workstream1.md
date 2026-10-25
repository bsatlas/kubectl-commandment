# Workstream 1 compatibility ledger

This ledger is frozen before implementation. `PRESERVE_EXACT` means byte/process behavior is part of the first-workstream oracle. `PRESERVE_SEMANTIC` means the same user-visible or state transition meaning is required while non-contractual presentation details may differ. `OUT_OF_FIRST_WORKSTREAM` means the behavior is explicitly retained for later compatibility closure rather than silently dropped.

## Root

| Subject | Disposition | First-workstream contract |
| --- | --- | --- |
| executable name | PRESERVE_EXACT | `kubectl` |
| root flag order | PRESERVE_SEMANTIC | selected root/global flags remain usable in kubectl's normal interspersed positions |
| root `--kubeconfig` | PRESERVE_SEMANTIC | selects explicit kubeconfig path for child operations |
| root `--context` | PRESERVE_SEMANTIC | overrides kubeconfig current context for child operations |
| root `-n`, `--namespace` | PRESERVE_SEMANTIC | supplies namespace selection to resource operations |
| complete kubectl global flag inventory | OUT_OF_FIRST_WORKSTREAM | port after style closure by coherent root-family workstream |
| `.kuberc` preferences | OUT_OF_FIRST_WORKSTREAM | later dedicated dynamic/preferences proof |
| plugin materialization | OUT_OF_FIRST_WORKSTREAM | later dedicated plugin proof |

## `config use-context`

| Subject | Disposition | First-workstream contract |
| --- | --- | --- |
| command path | PRESERVE_EXACT | `kubectl config use-context CONTEXT_NAME` |
| alias | PRESERVE_EXACT | `kubectl config use CONTEXT_NAME` |
| positional grammar | PRESERVE_EXACT | exactly one context-name operand |
| context-name completion | PRESERVE_SEMANTIC | candidates come from the selected kubeconfig, not a static list |
| selected kubeconfig | PRESERVE_SEMANTIC | same kubeconfig source/path meaning in deterministic fixture |
| unknown context | PRESERVE_SEMANTIC | rejected before mutation |
| successful mutation | PRESERVE_SEMANTIC | set `CurrentContext` and persist through Kubernetes clientcmd behavior |
| success stdout | PRESERVE_EXACT | `Switched to context %q.\n` |
| ordinary filesystem/config errors | PRESERVE_SEMANTIC | preserve cause and failure outcome; exact Cobra wrapper prose is not frozen here |

## `get` declared input surface

The complete current `NewCmdGet` local declared input surface is in Workstream 1. Root/global namespace/context/kubeconfig remain root-owned and are consumed by the child rather than redeclared.

```text
resource/name operands
-o, --output
--no-headers
--show-managed-fields
--template
--allow-missing-template-keys
--show-labels
--sort-by
-L, --label-columns
--show-kind
--raw
-w, --watch
--watch-only
--output-watch-events
--ignore-not-found
--field-selector
-A, --all-namespaces
--server-print
-f, --filename
-k, --kustomize
-R, --recursive
--chunk-size
-l, --selector
--subresource
```

The current output-format domain is also preserved for admission/completion:

```text
json yaml kyaml name go-template go-template-file template templatefile
jsonpath jsonpath-as-json jsonpath-file custom-columns custom-columns-file wide
```

## `get` policy rows

| Subject | Disposition | First-workstream contract |
| --- | --- | --- |
| `--raw` with resource/name operands | PRESERVE_SEMANTIC | reject |
| `--raw` with watch/watch-only | PRESERVE_SEMANTIC | reject |
| `--raw` with selector | PRESERVE_SEMANTIC | reject |
| `--raw` with output | PRESERVE_SEMANTIC | reject |
| malformed `--raw` request URI | PRESERVE_SEMANTIC | reject |
| `--show-labels` with nonempty non-`wide` output | PRESERVE_SEMANTIC | reject |
| label columns with custom-columns output | PRESERVE_SEMANTIC | reject |
| `--output-watch-events` without watch/watch-only | PRESERVE_SEMANTIC | reject |
| `-k` with filename input | PRESERVE_SEMANTIC | reject |
| `-R` without filename input | PRESERVE_SEMANTIC | allowed by current `FilenameOptions`; `-R` is rejected with `-k`, not merely because `-f` is absent |
| effective namespace/all-namespaces | PRESERVE_SEMANTIC | derived from root selection + kubeconfig through Kubernetes loader |
| custom output disables server print | PRESERVE_SEMANTIC | preserve current `Complete` meaning |
| `--chunk-size` numeric domain | PRESERVE_SEMANTIC | signed 64-bit kubectl range and default `500` |

## `get` runtime rows

| Subject | Disposition | First-workstream contract |
| --- | --- | --- |
| representative namespaced read | PRESERVE_SEMANTIC | real Kubernetes REST/discovery/resource-builder path, not canned objects |
| representative JSON/YAML/name output | PRESERVE_SEMANTIC | real Kubernetes printer path |
| raw GET | PRESERVE_SEMANTIC | real Kubernetes REST client path |
| resource completion | PRESERVE_SEMANTIC | Kubernetes discovery-backed candidates without package-global broad Factory |
| all watch/stream edge cases | OUT_OF_FIRST_WORKSTREAM | later get-family runtime closure |
| exhaustive table/server-print byte parity | OUT_OF_FIRST_WORKSTREAM | later get-family presentation closure |
| exhaustive sorting/custom-column/template byte parity | OUT_OF_FIRST_WORKSTREAM | declared/admitted now; exhaustive runtime parity later |
| underscore-to-hyphen flag normalization | OUT_OF_FIRST_WORKSTREAM | explicit compatibility row; current Commandment has no equivalent built-in normalizer at this baseline |
| Cobra `SuggestFor` hints for `get` (`list`, `ls`) | OUT_OF_FIRST_WORKSTREAM | legacy rejects these spellings and only offers typo suggestions; they are not aliases and must not become accepted Commandment command paths |

## Closure status

The repository-owned differential oracle is `compare_test.go`. It invokes the exact legacy kubectl and Commandment candidate as separate processes against equivalent deterministic fixtures. Workstream 1 currently proves:

- exact stdout plus independently parsed kubeconfig state for `config use-context`;
- exact stdout for `config current-context` (retained mutation pressure);
- exact stdout for `get --raw /healthz`;
- structured resource identity parity for `get pods -o json`;
- equivalent nonzero/empty-stdout rejection for the selected semantic policy rows;
- semantic completion separately through the public Commandment `Complete` operation and Kubernetes-backed discovery/resource fixtures.

Error prose for semantic policy rows is deliberately not byte-frozen in Workstream 1. Legacy Cobra validation messages and Commandment diagnostics differ while the selected rejection contract agrees. Exhaustive `get` runtime/presentation, watch behavior, complete root globals, `.kuberc`, plugins, and flag normalization remain explicitly downstream.
