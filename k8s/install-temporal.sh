#!/bin/bash
set -e

# Standalone Temporal install/upgrade for the cosmic-cluster (referenced by
# TEMPORAL_SETUP.md; setup-cluster.sh delegates here). Idempotent.
#
# Deploys straight from the COMMITTED temporal-values.yaml.template and
# injects the RDS endpoint via --set (onejump's proven pattern) — there is
# deliberately no hand-maintained local temporal-values.yaml: a stale local
# copy predating the chart v1.0 layout is exactly what broke the last
# upgrade ("'cassandra' is no longer a supported top-level key").
#
# The RDS endpoint comes from POSTGRES_HOST in the focus/temporal
# Secrets Manager secret, overridable with the RDS_ENDPOINT env var. The DB
# password is mirrored from the same secret into the k8s Secret
# temporal-postgres-credentials that the values reference via existingSecret.

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

cd "$(dirname "$0")"

# The kubeconfig lives at the repo root (../kubeconfig.yaml, gitignored) so
# nothing touches ~/.kube. Write it with:
#   aws eks update-kubeconfig --name cosmic-cluster --region us-east-1 \
#     --profile cosmic --kubeconfig kubeconfig.yaml
KUBECONFIG_FILE="${KUBECONFIG_FILE:-../kubeconfig.yaml}"
if [[ ! -f ${KUBECONFIG_FILE} ]]; then
	echo -e "${RED}Error: ${KUBECONFIG_FILE} not found!${NC}"
	echo "From the repo root run:"
	echo "  aws eks update-kubeconfig --name cosmic-cluster --region us-east-1 --profile cosmic --kubeconfig kubeconfig.yaml"
	exit 1
fi
export KUBECONFIG="${KUBECONFIG_FILE}"

echo -e "${YELLOW}Checking kubectl configuration...${NC}"
if ! kubectl cluster-info &>/dev/null; then
	echo -e "${RED}Error: kubectl is not configured or cluster is not accessible${NC}"
	exit 1
fi
echo -e "${GREEN}✓ kubectl configured${NC}"

echo -e "${BLUE}Reading focus/temporal from Secrets Manager...${NC}"
TEMPORAL_SM=$(aws secretsmanager get-secret-value --secret-id focus/temporal \
	--query SecretString --output text --profile cosmic)
TEMPORAL_PW=$(jq -r .POSTGRES_PWD <<<"${TEMPORAL_SM}")
RDS_ENDPOINT="${RDS_ENDPOINT:-$(jq -r '.POSTGRES_HOST // empty' <<<"${TEMPORAL_SM}")}"

# POSTGRES_HOST must be a bare hostname. A full URL (postgresql://user:pw@host:5432/db)
# would make the chart split "host:5432" at the wrong colon and Temporal would
# try to reach host "postgresql" on port "//user" — repair the common mistake
# and refuse anything that still isn't a hostname.
if [[ ${RDS_ENDPOINT} == *[:/@]* ]]; then
	echo -e "${YELLOW}POSTGRES_HOST looks like a connection string; extracting the hostname.${NC}"
	RDS_ENDPOINT="${RDS_ENDPOINT#*://}" # strip scheme
	RDS_ENDPOINT="${RDS_ENDPOINT##*@}"  # strip user:password@
	RDS_ENDPOINT="${RDS_ENDPOINT%%/*}"  # strip /database?params
	RDS_ENDPOINT="${RDS_ENDPOINT%%:*}"  # strip :port
fi
if [[ -n ${RDS_ENDPOINT} && ${RDS_ENDPOINT} == *[:/@]* ]]; then
	echo -e "${RED}Error: RDS endpoint must be a bare hostname, got '${RDS_ENDPOINT}'.${NC}"
	exit 1
fi

if [[ -z ${RDS_ENDPOINT} ]]; then
	echo -e "${RED}Error: RDS endpoint unknown.${NC}"
	echo "Either add POSTGRES_HOST to the focus/temporal secret in Secrets"
	echo "Manager, or run with: RDS_ENDPOINT=<host> ./install-temporal.sh"
	exit 1
fi
echo -e "${GREEN}✓ RDS endpoint: ${RDS_ENDPOINT}${NC}"

# The DB password is mirrored into a k8s Secret the values file references via
# existingSecret. We own this Secret so it survives interrupted/rolled-back
# upgrades (unlike the chart's inline-password hook Secret).
echo -e "${BLUE}Mirroring Temporal DB password into the cluster...${NC}"
kubectl get namespace focus-temporal >/dev/null 2>&1 ||
	kubectl create namespace focus-temporal
SECRET_YAML=$(kubectl create secret generic temporal-postgres-credentials \
	--namespace focus-temporal \
	--from-literal=password="${TEMPORAL_PW}" \
	--dry-run=client -o yaml)
kubectl apply -f - <<<"${SECRET_YAML}"

echo -e "${BLUE}Adding Helm repositories...${NC}"
helm repo add temporalio https://go.temporal.io/helm-charts 2>/dev/null || true
helm repo update
echo -e "${GREEN}✓ Helm repositories updated${NC}"
echo ""

echo -e "${BLUE}Installing Temporal in focus-temporal namespace...${NC}"
# Pin the chart version. 1.4.0 (server 1.31.1) is the version deployed and
# healthy across the fleet; admintools 1.31 in the values file is compatible.
# Do NOT downgrade a live cluster — Temporal schema migrations don't reverse.
helm upgrade --install temporal temporalio/temporal \
	--namespace focus-temporal \
	--create-namespace \
	--version 1.4.0 \
	--values temporal-values.yaml.template \
	--set server.config.persistence.datastores.default.sql.connectAddr="${RDS_ENDPOINT}:5432" \
	--set server.config.persistence.datastores.visibility.sql.connectAddr="${RDS_ENDPOINT}:5432" \
	--wait \
	--timeout 10m
echo -e "${GREEN}✓ Temporal ready${NC}"
echo ""

# Placement report (Graviton migration): server/web pods should be on arm64
# nodes; admintools is deliberately unpinned and may run on either pool.
echo -e "${BLUE}Pod placement:${NC}"
kubectl get pods -n focus-temporal -o wide
echo ""
NODES=$(kubectl get pods -n focus-temporal \
	-o jsonpath='{range .items[?(@.status.phase=="Running")]}{.spec.nodeName}{"\n"}{end}' | sort -u || true)
for node in ${NODES}; do
	ARCH=$(kubectl get node "${node}" -o jsonpath='{.metadata.labels.kubernetes\.io/arch}')
	echo "node ${node} arch=${ARCH}"
done
