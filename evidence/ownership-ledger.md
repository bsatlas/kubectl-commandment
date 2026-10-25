# Workstream 1 ownership ledger

The source translation does not preserve kubectl's incumbent `GetOptions` decomposition merely because it exists. Responsibilities are assigned by semantic owner.

| Current kubectl responsibility | Workstream 1 owner |
| --- | --- |
| command path, aliases, argument/flag grammar | Commandment declarations |
| output-format accepted language/completion set | Commandment value language + completion declaration |
| incompatible input combinations / admission | Commandment rules where representable without duplicating parsing |
| finalized root kubeconfig/context/namespace values | root Commandment destinations, accessed explicitly by child execution/completion |
| context-name validation and completion | dynamic Commandment domain backed by Kubernetes kubeconfig service |
| effective namespace calculation | Kubernetes clientcmd loader, after Commandment admission |
| discovery / REST mapper / REST config | Kubernetes client-go / cli-runtime services |
| resource building | Kubernetes cli-runtime `resource.Builder` |
| object/table printing | Kubernetes printers / kubectl printer machinery |
| raw REST operation | Kubernetes REST client + kubectl raw HTTP helper |
| streams / working process facilities | Commandment execution access, adapted to Kubernetes IOStreams at operation boundary |
| Cobra/pflag post-parse rereads | removed; no owner in new frontend |
| broad `cmdutil.Factory` as completion/application state bus | removed from new semantic path; replaced by narrow ordinary Go services |
| Cobra command traversal for tests/tooling | removed; Commandment compiled application is the query subject |

## Donor/import law

Frontend-neutral Kubernetes packages are imported directly. A current Kubernetes package may remain a transitive Cobra/pflag dependency when reused for frontend-neutral application behavior, but the new frontend may not construct, parse, traverse, mutate, execute, or use a Cobra/pflag graph as semantic authority.
