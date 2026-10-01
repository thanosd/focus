# Production setup checklist

Everything that must exist before the first `git push` to `main` deploys
Focus. Items are ordered so each step has what it needs. Names are exact —
the workflows and chart reference them literally.

Target: `focus.cosmicteacups.com` (app) and `focus-api.cosmicteacups.com`
(API + MCP), namespace `focus` on `cosmic-cluster`, production only.

## 1. GitHub repository

- [ ] Create `thanosd/focus` (private), push this repo, default branch `main`.
- [ ] Branch protection on `main` requiring the `CI` checks (optional but
      matches team-metrics).
- No repository secrets are needed: the deploy workflow authenticates to AWS
  with OIDC and reads everything else from Secrets Manager.

## 2. Google OAuth client

In <https://console.cloud.google.com/apis/credentials> (any GCP project):

- [ ] OAuth consent screen: **Internal** if the `cosmicteacups.com`
      workspace, else External + add your account as a test user.
- [ ] Create **OAuth client ID → Web application** named `Focus`.
  - Authorized JavaScript origins: `https://focus.cosmicteacups.com`
  - Authorized redirect URIs:
    `https://focus-api.cosmicteacups.com/api/auth/google/callback`
    (and `http://localhost:8000/api/auth/google/callback` for local dev)
- [ ] Note the **Client ID** and **Client secret** → `GOOGLE_CLIENT_ID`,
      `GOOGLE_CLIENT_SECRET` below.

## 3. Claude API key (optional but recommended)

- [ ] Create a key in the Anthropic Console for the project → `CLAUDE_API_KEY`.
      Without it, deferral still works for every phrase the rule parser
      knows; only unusual phrases ("after the board meeting") fail with a 400.

## 4. AWS resources (Pulumi)

```bash
cd infra && npm install && pulumi login
pulumi stack init cosmic-teacups-org/focus-infra/production
pulumi config set aws:region us-east-1 && pulumi config set aws:profile cosmic
AWS_PROFILE=cosmic pulumi up
```

Creates, in account `131925870818` / `us-east-1`:

- [ ] ECR repos `focus-backend`, `focus-frontend`
- [ ] IAM role `focus-deploy` (GitHub OIDC trust for `thanosd/focus`) with
      ECR push, `secretsmanager:GetSecretValue` on the app secret, and
      `eks:DescribeCluster`
- [ ] EKS access entry + ClusterRoleBinding for `focus-deploy`
- [ ] Secrets Manager secret `cosmic/focus/production` (empty value)
- [ ] Namespace `focus` and ServiceAccount `focus`
- [ ] Route 53 CNAMEs `focus.cosmicteacups.com` and
      `focus-api.cosmicteacups.com` → ingress-nginx NLB

If you prefer to create these by hand, those are the exact names the
workflow and chart expect (`AWS_ROLE_ARN` is hard-coded in
`.github/workflows/deploy-production.yml`).

## 5. Databases on the shared RDS

Over Tailscale, with the `cosmic` AWS profile:

```bash
./deploy/bootstrap-db.sh <rds-endpoint>
```

Creates roles/databases `focus_production`, `focus_temporal`,
`focus_temporal_visibility` and prints the generated passwords. Keep them
for the next two steps.

## 6. Secrets Manager: `cosmic/focus/production`

One JSON object. The deploy workflow camelCases each key into Helm values
(`DATABASE_URL` → `databaseUrl`). Required keys:

| Key                                  | Value                                                                                                  |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `DATABASE_URL`                       | from bootstrap-db.sh: `postgresql://focus_production:<pw>@<rds>:5432/focus_production?sslmode=require` |
| `GOOGLE_CLIENT_ID`                   | from step 2                                                                                            |
| `GOOGLE_CLIENT_SECRET`               | from step 2                                                                                            |
| `AUTH_ALLOWED_EMAILS`                | `thanos.diacakis@cosmicteacups.com` (comma-separate to add more)                                       |
| `CSRF_AUTH_KEY`                      | `openssl rand -base64 32`                                                                              |
| `NEXT_SERVER_ACTIONS_ENCRYPTION_KEY` | `openssl rand -base64 32` — must stay stable across deploys                                            |
| `CLAUDE_API_KEY`                     | from step 3 (may be an empty string)                                                                   |

```bash
aws secretsmanager put-secret-value --profile cosmic \
  --secret-id cosmic/focus/production \
  --secret-string '{"DATABASE_URL":"...","GOOGLE_CLIENT_ID":"...","GOOGLE_CLIENT_SECRET":"...","AUTH_ALLOWED_EMAILS":"thanos.diacakis@cosmicteacups.com","CSRF_AUTH_KEY":"...","NEXT_SERVER_ACTIONS_ENCRYPTION_KEY":"...","CLAUDE_API_KEY":"..."}'
```

Non-secret settings (`CORS_ORIGINS`, `FRONTEND_URL`, `API_URL`,
`NEXT_PUBLIC_API_URL`) are set in the workflow's `env:` block, not here.

## 7. Secrets Manager: `focus/temporal`

Used only by `k8s/install-temporal.sh`:

```bash
aws secretsmanager create-secret --profile cosmic --name focus/temporal \
  --secret-string '{"POSTGRES_HOST":"<rds-endpoint>","POSTGRES_USER":"focus_temporal","POSTGRES_PWD":"<pw from bootstrap-db.sh>"}'
```

## 8. Temporal

```bash
aws eks update-kubeconfig --name cosmic-cluster --region us-east-1 --profile cosmic --kubeconfig kubeconfig.yaml
./k8s/install-temporal.sh
```

Installs the `temporalio/temporal` chart (pinned 1.4.0) into
`focus-temporal` against the two `focus_temporal*` databases. The worker
auto-registers the `focus` Temporal namespace and the nightly schedule.

## 9. First deploy

- [ ] Push to `main` (or run **Deploy to Production** manually). The
      pre-upgrade Job runs `focus migrate`; `--atomic` rolls back on failure.
- [ ] Watch: `kubectl get pods -n focus -w`
- [ ] Open <https://focus.cosmicteacups.com>, sign in with Google.
- [ ] Settings → create an API token → `claude mcp add --transport http focus https://focus-api.cosmicteacups.com/mcp --header "Authorization: Bearer fcs_..."`

## 10. Import your OmniFocus data (optional)

See [../import-omnifocus.md](../import-omnifocus.md): dry-run first, then run
`focus import-omnifocus` against the production `DATABASE_URL` over
Tailscale or via `kubectl exec` into the backend pod.

## Summary of named things

| Kind                | Name                                                                    |
| ------------------- | ----------------------------------------------------------------------- |
| GitHub repo         | `thanosd/focus`                                                         |
| Google OAuth client | redirect `https://focus-api.cosmicteacups.com/api/auth/google/callback` |
| IAM role            | `arn:aws:iam::131925870818:role/focus-deploy`                           |
| ECR                 | `focus-backend`, `focus-frontend`                                       |
| Secrets Manager     | `cosmic/focus/production`, `focus/temporal`                             |
| RDS databases       | `focus_production`, `focus_temporal`, `focus_temporal_visibility`       |
| K8s namespaces      | `focus`, `focus-temporal`                                               |
| ServiceAccount      | `focus`                                                                 |
| DNS                 | `focus.cosmicteacups.com`, `focus-api.cosmicteacups.com`                |
| TLS secret          | `focus-tls` (cert-manager, `letsencrypt-prod`)                          |
