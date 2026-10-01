import * as aws from "@pulumi/aws";
import * as k8s from "@pulumi/kubernetes";
import * as pulumi from "@pulumi/pulumi";

// ============================================================================
// CONFIGURATION
// ============================================================================
// Mirrors team-metrics/infra: this stack owns only the app-specific AWS
// resources. The EKS cluster, RDS, VPC, ingress-nginx and cert-manager are
// provisioned by the cosmic-k8s-cluster stack.

const config = new pulumi.Config();

const appHostname = config.get("appHostname") || "focus.cosmicteacups.com";
const apiHostname = config.get("apiHostname") || "focus-api.cosmicteacups.com";
const githubOrg = config.get("githubOrg") || "thanosd";
const githubRepo = config.get("githubRepo") || "focus";
const cosmicClusterName = config.get("cosmicClusterName") || "cosmic-cluster";
const namespaceName = config.get("namespace") || "focus";

// ============================================================================
// DATA LOOKUPS
// ============================================================================

const githubOidc = aws.iam.getOpenIdConnectProviderOutput({
  url: "https://token.actions.githubusercontent.com",
});

const cosmicCluster = aws.eks.getClusterOutput({ name: cosmicClusterName });

// ============================================================================
// IAM DEPLOY ROLE (GitHub Actions OIDC)
// ============================================================================

const deployRole = new aws.iam.Role("focus-deploy-role", {
  name: "focus-deploy",
  assumeRolePolicy: githubOidc.arn.apply(
    (oidcArn: string) =>
      aws.iam.getPolicyDocumentOutput({
        statements: [
          {
            actions: ["sts:AssumeRoleWithWebIdentity"],
            principals: [{ type: "Federated", identifiers: [oidcArn] }],
            conditions: [
              {
                test: "StringEquals",
                variable: "token.actions.githubusercontent.com:aud",
                values: ["sts.amazonaws.com"],
              },
              {
                test: "StringLike",
                variable: "token.actions.githubusercontent.com:sub",
                values: [`repo:${githubOrg}/${githubRepo}:*`],
              },
            ],
          },
        ],
      }).json,
  ),
  tags: { Name: "focus-deploy", Project: "focus" },
});

new aws.iam.RolePolicy("focus-eks-policy", {
  role: deployRole.name,
  policy: cosmicCluster.arn.apply(
    (clusterArn: string) =>
      aws.iam.getPolicyDocumentOutput({
        statements: [
          { actions: ["eks:DescribeCluster"], resources: [clusterArn] },
        ],
      }).json,
  ),
});

// ============================================================================
// ECR REPOSITORIES
// ============================================================================

const lifecyclePolicy = JSON.stringify({
  rules: [
    {
      rulePriority: 1,
      description: "Keep last 10 images",
      selection: {
        tagStatus: "any",
        countType: "imageCountMoreThan",
        countNumber: 10,
      },
      action: { type: "expire" },
    },
  ],
});

const backendRepo = new aws.ecr.Repository("focus-backend", {
  name: "focus-backend",
  imageTagMutability: "MUTABLE",
  imageScanningConfiguration: { scanOnPush: true },
  tags: { Project: "focus" },
});

const frontendRepo = new aws.ecr.Repository("focus-frontend", {
  name: "focus-frontend",
  imageTagMutability: "MUTABLE",
  imageScanningConfiguration: { scanOnPush: true },
  tags: { Project: "focus" },
});

new aws.ecr.LifecyclePolicy("focus-backend-lifecycle", {
  repository: backendRepo.name,
  policy: lifecyclePolicy,
});
new aws.ecr.LifecyclePolicy("focus-frontend-lifecycle", {
  repository: frontendRepo.name,
  policy: lifecyclePolicy,
});

new aws.iam.RolePolicy("focus-ecr-policy", {
  role: deployRole.name,
  policy: pulumi.all([backendRepo.arn, frontendRepo.arn]).apply(
    ([backendArn, frontendArn]) =>
      aws.iam.getPolicyDocumentOutput({
        statements: [
          {
            actions: [
              "ecr:GetDownloadUrlForLayer",
              "ecr:BatchGetImage",
              "ecr:BatchCheckLayerAvailability",
              "ecr:PutImage",
              "ecr:InitiateLayerUpload",
              "ecr:UploadLayerPart",
              "ecr:CompleteLayerUpload",
            ],
            resources: [backendArn, frontendArn],
          },
          { actions: ["ecr:GetAuthorizationToken"], resources: ["*"] },
        ],
      }).json,
  ),
});

// ============================================================================
// EKS ACCESS
// ============================================================================
// Access Entry + aws-auth-style ClusterRoleBinding, same belt-and-suspenders
// approach as team-metrics (the cluster's auth mode ignored Access Entries).

new aws.eks.AccessEntry("focus-access-entry", {
  clusterName: cosmicClusterName,
  principalArn: deployRole.arn,
  type: "STANDARD",
  userName: "focus-deploy",
});

new aws.eks.AccessPolicyAssociation("focus-access-policy", {
  clusterName: cosmicClusterName,
  principalArn: deployRole.arn,
  policyArn:
    "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy",
  accessScope: { type: "cluster" },
});

// ============================================================================
// SECRETS MANAGER
// ============================================================================
// One JSON secret holding every runtime setting the deploy workflow
// injects into the Helm release. The value is NOT managed here — populate
// it by hand after `pulumi up`; see docs/deploy/setup-checklist.md.

const appSecret = new aws.secretsmanager.Secret("focus-secret", {
  name: "cosmic/focus/production",
  description: "Application secrets for Focus (production)",
  tags: {
    Name: "focus-production",
    Project: "focus",
    Environment: "production",
  },
});

new aws.iam.RolePolicy("focus-secrets-policy", {
  role: deployRole.name,
  policy: appSecret.arn.apply(
    (arn) => `{
    "Version": "2012-10-17",
    "Statement": [
      { "Effect": "Allow", "Action": "secretsmanager:GetSecretValue", "Resource": "${arn}" },
      { "Effect": "Allow", "Action": "kms:Decrypt", "Resource": "*" }
    ]
  }`,
  ),
});

// ============================================================================
// KUBERNETES PROVIDER + NAMESPACE + SERVICE ACCOUNT
// ============================================================================

const kubeconfig = pulumi
  .all([
    cosmicCluster.name,
    cosmicCluster.endpoint,
    cosmicCluster.certificateAuthorities,
  ])
  .apply(([name, endpoint, cas]) => {
    const caData = cas[0]?.data ?? "";
    return `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${caData}
    server: ${endpoint}
  name: ${name}
contexts:
- context:
    cluster: ${name}
    user: ${name}
  name: ${name}
current-context: ${name}
kind: Config
users:
- name: ${name}
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
        - "eks"
        - "get-token"
        - "--cluster-name"
        - "${name}"
`;
  });

const k8sProvider = new k8s.Provider("k8s-provider", { kubeconfig });

const namespace = new k8s.core.v1.Namespace(
  "focus-ns",
  { metadata: { name: namespaceName } },
  { provider: k8sProvider },
);

const serviceAccount = new k8s.core.v1.ServiceAccount(
  "focus-sa",
  { metadata: { name: "focus", namespace: namespace.metadata.name } },
  { provider: k8sProvider },
);

new k8s.rbac.v1.ClusterRoleBinding(
  "focus-deploy-cluster-admin",
  {
    metadata: { name: "focus-deploy-cluster-admin" },
    subjects: [
      {
        kind: "User",
        name: "focus-deploy",
        apiGroup: "rbac.authorization.k8s.io",
      },
    ],
    roleRef: {
      kind: "ClusterRole",
      name: "cluster-admin",
      apiGroup: "rbac.authorization.k8s.io",
    },
  },
  { provider: k8sProvider },
);

// ============================================================================
// DNS — point both hostnames at the shared ingress-nginx NLB
// ============================================================================
// Same pattern as Cap in the cluster stack: look up the ingress-nginx
// controller Service's load balancer hostname and CNAME to it in the
// cosmicteacups.com hosted zone.

const zoneName = appHostname.split(".").slice(1).join(".");
const zone = aws.route53.getZoneOutput({ name: zoneName });

const ingressService = k8s.core.v1.Service.get(
  "ingress-nginx-controller",
  "ingress-nginx/ingress-nginx-controller",
  { provider: k8sProvider },
);
const ingressNlbHostname = ingressService.status.apply(
  (status) => status.loadBalancer.ingress[0].hostname,
);

new aws.route53.Record("focus-app-dns", {
  zoneId: zone.zoneId,
  name: appHostname,
  type: "CNAME",
  ttl: 300,
  records: [ingressNlbHostname],
  allowOverwrite: true,
});

new aws.route53.Record("focus-api-dns", {
  zoneId: zone.zoneId,
  name: apiHostname,
  type: "CNAME",
  ttl: 300,
  records: [ingressNlbHostname],
  allowOverwrite: true,
});

// ============================================================================
// EXPORTS
// ============================================================================

export const deployRoleArn = deployRole.arn;
export const secretArn = appSecret.arn;
export const secretName = appSecret.name;
export const backendRepoUrl = backendRepo.repositoryUrl;
export const frontendRepoUrl = frontendRepo.repositoryUrl;
export const serviceAccountName = serviceAccount.metadata.name;
export const appUrl = `https://${appHostname}`;
export const apiUrl = `https://${apiHostname}`;
