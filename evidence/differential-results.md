# Workstream 1 differential results

## Subjects

```text
legacy kubectl source     3930cfbbb185e71ecb7b993a5582de6f775bc1f0
kubectl source commit    aac6731c524e03f6039d79be1b52a2bb0d799700
kubectl source tree      c9d2c1871faf87b318766648c83700b0541b375d
Commandment baseline     a02adf48fc503a57a0898caf235f800b08205270
Commandment tested donor 73621550f7ceba1b58c9c71d92a669580beec981
Commandment donor tree   595d71d81bdff0788613e21bff715b90372d5bf0
candidate binary sha256  bf98b66f086313e584f42207773368e939ec8b57b6b3add59989b1647862ba1d
```

The comparison binaries are separate processes. The legacy binary is the current Cobra/pflag kubectl baseline. The candidate binary is the Commandment frontend using real Kubernetes kubeconfig, REST/discovery, resource-builder, raw HTTP, and printer machinery.

The repository-owned oracle is `compatibility/compare_test.go`. It requires `KUBECTL_LEGACY` and `KUBECTL_COMMANDMENT`; without those explicit executable inputs it skips rather than pretending to prove differential behavior.

## Returned differential run

The exact kubectl source commit above, composed for verification with the exact Commandment donor above, returned:

```text
use_context                              PASS
current_context                          PASS
raw_get                                  PASS
json_resource_read                       PASS
namespace_override                       PASS
context_override                         PASS
all_namespaces                           PASS
chunk_size_int64_max                     PASS
reject_raw_with_resources                PASS
reject_raw_with_watch                    PASS
reject_raw_with_selector                 PASS
reject_raw_with_output                   PASS
reject_show_labels_with_json             PASS
reject_watch_events_without_watch        PASS
reject_filename_and_kustomize            PASS
reject_recursive_and_kustomize           PASS
reject_missing_input                     PASS
reject_chunk_size_int64_overflow         PASS
reject_get_suggest-for_list_is_not_alias PASS
reject_get_suggest-for_ls_is_not_alias   PASS
```

## Comparison strength

| Case | Strength | Independent observation |
| --- | --- | --- |
| `config use-context dev` | EXACT_BYTES + STRUCTURED | exact stdout; `clientcmd.LoadFromFile` independently confirms persisted `CurrentContext=dev` in each fixture |
| `config current-context` | EXACT_BYTES | process stdout |
| `get --raw /healthz` | EXACT_BYTES | process stdout against the same HTTP fixture |
| `get pods -o json` | STRUCTURED | JSON is independently decoded and normalized to resource kind/name/namespace; formatting and empty list-metadata details are not treated as the semantic contract |
| root `--namespace other` override | STRUCTURED | both frontends select the same namespaced resource observation from the override |
| root `--context other` override | STRUCTURED | both frontends select the same namespaced resource observation through the alternate kubeconfig context |
| `--all-namespaces` | STRUCTURED | both frontends observe the same cross-namespace resource set |
| `get pods --chunk-size 9223372036854775807 -o json` | STRUCTURED + INPUT RANGE | both frontends accept the signed 64-bit maximum and produce the same resource observation |
| `--chunk-size 9223372036854775808` | SEMANTIC | both frontends reject one-above-max before successful execution |
| `list pods` / `ls pods` | SEMANTIC | both frontends reject the spellings; legacy Cobra `SuggestFor` metadata is not promoted into executable aliases |
| selected invalid `get` combinations | SEMANTIC / EXACT_PROCESS subset | both processes reject with nonzero status and empty stdout; stderr prose is intentionally not byte-frozen |

The JSON case exposes one harmless presentation-level difference in the synthetic fixture: legacy output includes an empty list `metadata.resourceVersion`, while the candidate generic list projection omits that empty field. Workstream 1 classified JSON output as semantic, not byte-exact, so the independent oracle compares the resource observation rather than treating an empty metadata field as a migration defect.

## Difference classification

The selected invalid-input cases use different diagnostic text. This is `OLD_IMPLEMENTATION_DETAIL` for Workstream 1, not permission to ignore error behavior generally: the ledger freezes rejection outcome and stream behavior semantically, while exhaustive diagnostic presentation is a later compatibility subject.

No first-workstream preserved row has an unresolved `BUG`, `COMMANDMENT_GAP`, or `ORACLE_DEFECT` classification on the exact source candidate.
