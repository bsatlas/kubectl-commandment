# Workstream 1 performance smoke evidence

Classification: **SUPPORTING**, not `DECISION-GRADE`.

This is an early blocker check only; it predates the final `aac6731` source candidate. The functional candidate is intentionally incomplete relative to full kubectl topology, caches are warm, and no fresh-cache build protocol was completed. These numbers must not be used as a final SIG CLI performance claim.

## Subjects

```text
legacy kubectl source   3930cfbbb185e71ecb7b993a5582de6f775bc1f0
candidate executable    built from b7d92b5 pre-closure source checkpoint
Go                      1.27.1 linux/amd64
```

Equivalent process invocations used the same deterministic kubeconfig/local API fixture. Each workload received warmup runs before measurement.

| Workload | Legacy median | Commandment median | Legacy p95 | Commandment p95 |
| --- | ---: | ---: | ---: | ---: |
| `config current-context` (30 runs) | 26.44 ms | 22.21 ms | 31.89 ms | 23.72 ms |
| `get pods -o json` local fixture (15 runs) | 28.69 ms | 25.33 ms | 32.46 ms | 27.37 ms |
| root Help (15 runs) | 26.23 ms | 22.00 ms | 26.92 ms | 24.24 ms |
| `config` Help (15 runs) | 27.28 ms | 21.21 ms | 29.88 ms | 24.38 ms |

Help measurements are timing smoke only: the candidate intentionally has a smaller command tree, so the bytes/work performed are not equivalent enough for adoption conclusions.

Binary sizes from the same environment:

```text
legacy kubectl          106,103,736 bytes
kubectl-commandment      92,353,816 bytes
```

A warm-build comparison was attempted but the pristine Kubernetes transport's root `go.mod`/vendor replacement projection is not directly executable under the local standalone `go build -mod=vendor` form used for the candidate. Rather than mutate that evidence surface or report incomparable build numbers, build timing remains **NOT CLOSED** here.

## Disposition

No early startup/binary-size blocker is exposed by this partial functional candidate. This does **not** satisfy the plan's eventual scale/performance gate. Before a production-adoption-ready claim, run the preregistered decision-grade protocol against a kubectl-shape scale specimen/full candidate, including clean/fresh-cache build elapsed time and peak RSS, startup, completion, and scaling dimensions.
