# Kubernetes donor/import boundary

Workstream 1 reuses Kubernetes behavior without retaining Cobra/pflag as semantic authority.

| Subject | Disposition | Reason |
| --- | --- | --- |
| `k8s.io/client-go/tools/clientcmd` | `IMPORT_FRONTEND_NEUTRAL` | kubeconfig loading, mutation, and context state are Kubernetes application behavior independent of Commandment grammar |
| `k8s.io/cli-runtime/pkg/resource` | `IMPORT_FRONTEND_NEUTRAL` | resource builder and filename/resource selection execution remain Kubernetes application behavior |
| `k8s.io/cli-runtime/pkg/genericclioptions` | `IMPORT_FRONTEND_NEUTRAL` | REST client construction is consumed as an ordinary Kubernetes service, not as parser state |
| `k8s.io/cli-runtime/pkg/printers` | `IMPORT_FRONTEND_NEUTRAL` | object printing remains Kubernetes presentation/application machinery |
| `k8s.io/client-go/rest` and/apimachinery runtime/meta packages | `IMPORT_FRONTEND_NEUTRAL` | REST and API object behavior are domain/application dependencies |
| `k8s.io/kubectl/pkg/rawhttp` | `IMPORT_FRONTEND_NEUTRAL` | raw REST execution is reused directly |
| `k8s.io/kubectl/pkg/scheme` | `IMPORT_FRONTEND_NEUTRAL` | Kubernetes object decoding/type setting remains domain infrastructure |
| selected printer helpers from `k8s.io/kubectl/pkg/cmd/get` | `IMPORT_FRONTEND_NEUTRAL` | the consumed `PrintFlags` printer construction, `SortingPrinter`, and `TablePrinter` APIs are used after Commandment admission and do not receive or expose a Cobra command tree; the containing Kubernetes package is frontend-mixed and therefore still causes transitive Cobra/pflag dependencies |
| root/config/get grammar, value resolution, rules, completion ownership | `NEW_COMMANDMENT_FRONTEND` | authored directly against Commandment; no legacy parser/model participates |
| current Cobra kubectl executable | `LEGACY_ORACLE_ONLY` | used only as an external subprocess compatibility oracle |
| broader exported Kubernetes APIs whose signatures expose Cobra/pflag | `DEFER_UPSTREAM_API` | upstream ecosystem transition is a later SIG CLI adoption obligation and does not block the standalone Workstream-1 source experiment |

No Kubernetes production source is copied into the repository in Workstream 1, so `COPY_ADAPT_FRONTEND_TRAPPED` is not currently used.

## Transitive Cobra/pflag status

`go list -deps ./cmd/kubectl` on the exact source candidate includes:

```text
github.com/spf13/pflag
github.com/spf13/cobra
```

That is expected from reused frontend-mixed Kubernetes packages. Direct production imports in `kubectl-commandment` are **none**. The Commandment frontend does not construct, parse with, traverse, mutate, execute, or use a Cobra/pflag graph as a command/value/help/completion/testing authority.

This distinction is part of the migration evidence: dependency presence is not equivalent to semantic custody. A later upstream adoption program may choose to split or adapt frontend-trapped Kubernetes packages to remove the transitive dependency entirely, but Workstream 1 does not need to rewrite otherwise reusable Kubernetes printer/application behavior merely to make `go mod graph | grep cobra` empty.
