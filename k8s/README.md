# Kubernetes

- `focus/` — Helm chart: backend + frontend + worker Deployments, Services,
  Ingress (nginx + cert-manager), app Secret, and a pre-upgrade migration Job.
- `install-temporal.sh` + `temporal-values.yaml.template` — Temporal install
  into the `focus-temporal` namespace (see `TEMPORAL_SETUP.md`).

The deploy workflow renders `helm-values.yaml` (image tags, URLs) and
`secrets-values.yaml` (from AWS Secrets Manager) next to the chart and runs:

```bash
helm upgrade --install focus ./focus --namespace focus --create-namespace \
  --values focus/values.yaml --values helm-values.yaml --values secrets-values.yaml \
  --wait --timeout 5m --cleanup-on-fail --atomic
```

Lint locally with `helm lint k8s/focus --set secrets.databaseUrl=x`.
