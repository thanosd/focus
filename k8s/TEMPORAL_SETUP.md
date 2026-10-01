# Temporal Setup

Focus runs one Temporal worker (nightly maintenance: expired sessions and
sign-in nonces). It reuses the team-metrics recipe: the official
`temporalio/temporal` chart (pinned 1.4.0, server 1.31) with Postgres
persistence on the shared Cosmic RDS.

## Prerequisites

1. Databases `focus_temporal` and `focus_temporal_visibility` and the role
   `focus_temporal` — created by `deploy/bootstrap-db.sh`.
2. Secrets Manager secret `focus/temporal` with `POSTGRES_HOST`,
   `POSTGRES_USER`, `POSTGRES_PWD`.
3. `kubeconfig.yaml` at the repo root (gitignored):
   `aws eks update-kubeconfig --name cosmic-cluster --region us-east-1 --profile cosmic --kubeconfig kubeconfig.yaml`.

## Install / upgrade

```bash
./install-temporal.sh
```

The script mirrors the DB password into the `temporal-postgres-credentials`
Secret in `focus-temporal`, then installs from the committed
`temporal-values.yaml.template`, injecting the RDS endpoint with `--set`.
There is deliberately no hand-maintained `temporal-values.yaml`.

`numHistoryShards` (512) is fixed at cluster creation — never change it on
a live install.

## App wiring

The chart passes `TEMPORAL_HOST=temporal-frontend.focus-temporal.svc.cluster.local:7233`
and `TEMPORAL_NAMESPACE=focus` to the backend and worker
(`k8s/focus/values.yaml` → `worker.temporal`). The worker registers the
namespace and the `focus-maintenance` schedule (02:00 UTC) on start.
