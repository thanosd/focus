# Infrastructure (Pulumi, AWS)

Pulumi project that provisions the AWS-side bits Focus needs to run on
the shared Cosmic EKS cluster. **The cluster, RDS Postgres, VPC,
ingress-nginx and cert-manager are managed by the `cosmic-k8s-cluster`
stack** — this stack only owns app-specific resources.

## What this stack provisions

See `index.ts`:

- **ECR repositories** `focus-backend`, `focus-frontend` (10-image
  retention, scan-on-push)
- **IAM deploy role** `focus-deploy` with GitHub Actions OIDC trust
  scoped to `thanosd/focus`, allowed to push to those repos, read the
  app secret and describe the cluster
- **EKS access** for the deploy role (Access Entry + ClusterRoleBinding)
- **Secrets Manager** entry `cosmic/focus/production` (value populated by
  hand — see `docs/deploy/setup-checklist.md`)
- **Kubernetes namespace** `focus` and ServiceAccount `focus`
- **Route 53** CNAMEs `focus.cosmicteacups.com` and
  `focus-api.cosmicteacups.com` → the cluster's ingress-nginx NLB

## Running it

```bash
cd infra
npm install
pulumi login
pulumi stack init cosmic-teacups-org/focus-infra/production   # first time
pulumi config set aws:region us-east-1
pulumi config set aws:profile cosmic
export AWS_PROFILE=cosmic
pulumi up
```

Config keys (all optional): `appHostname`, `apiHostname`, `githubOrg`,
`githubRepo`, `githubOwnerId`, `githubRepoId` (for GitHub's immutable OIDC
subject, `repo:OWNER@ID/REPO@ID:*`), `cosmicClusterName`, `namespace`.

After `pulumi up`, set the GitHub Actions role ARN (output
`deployRoleArn`) matches `AWS_ROLE_ARN` in
`.github/workflows/deploy-production.yml` (it's hard-coded to
`arn:aws:iam::131925870818:role/focus-deploy`).
