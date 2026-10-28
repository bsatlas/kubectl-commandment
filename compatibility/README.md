# Differential compatibility

`compare_test.go` is the repository-owned subprocess differential oracle for behavior where comparing the legacy kubectl executable with the Commandment frontend is the strongest independent proof.

The test runs only when both executables are supplied:

```text
KUBECTL_LEGACY=/path/to/kubectl
KUBECTL_COMMANDMENT=/path/to/kubectl-commandment
```

The current deterministic fixture covers selected kubeconfig and `get` contracts, including context mutation/current-context output, raw reads, structured resource reads, namespace/context selection, all-namespaces behavior, large chunk-size admission, and selected invalid-input relationships.

Comparison strength is intentional per case. Exact stdout is used where presentation is part of the selected contract; structured resource observations are used where durable Kubernetes meaning is the contract. A difference is not automatically accepted or rejected solely because bytes differ.

This suite is not a second command registry and is not the completeness ledger for the frontend. Static command topology comes from the Commandment application itself. The differential suite remains deliberately narrower than the complete built-in command surface and should expand only when a behavior benefits from an independent legacy-process oracle rather than from Commandment semantic or Kubernetes-owned application tests.
