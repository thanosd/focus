#!/bin/bash
# Focus database bootstrap for the Cosmic cluster RDS.
# Run once over the Tailscale VPN with access to the RDS instance.
#
# Creates three databases on the shared RDS (prefixed "focus_" to avoid
# collisions with other projects):
#   1. focus_production           — application database
#   2. focus_temporal             — Temporal persistence
#   3. focus_temporal_visibility  — Temporal visibility
#
# Prerequisites: Tailscale connected, AWS CLI (profile: cosmic), psql.
#
# Usage:
#   ./bootstrap-db.sh <rds-endpoint>

set -e

if [[ -z $1 ]]; then
	echo "Usage: ./bootstrap-db.sh <rds-endpoint>"
	exit 1
fi

RDS_ENDPOINT="$1"
PROJECT="focus"

echo "🔑 Fetching master password from AWS Secrets Manager..."
MASTER_PASS=$(aws secretsmanager get-secret-value \
	--secret-id cosmic/db-password \
	--query SecretString --output text \
	--profile cosmic 2>/dev/null || true)

if [[ -z ${MASTER_PASS} ]]; then
	echo "⚠️  Could not fetch master password automatically."
	echo "   Enter the master postgres password manually:"
	read -rs MASTER_PASS
fi

APP_PASS=$(openssl rand -base64 24)
APP_PASS=${APP_PASS//[\/+=]/}

TEMPORAL_PASS=$(openssl rand -base64 24)
TEMPORAL_PASS=${TEMPORAL_PASS//[\/+=]/}

echo ""
echo "📦 Creating application database and user..."
PGPASSWORD="${MASTER_PASS}" psql -h "${RDS_ENDPOINT}" -U postgres -p 5432 <<EOSQL
DO \$\$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '${PROJECT}_production') THEN
    CREATE ROLE ${PROJECT}_production WITH LOGIN PASSWORD '${APP_PASS}';
  END IF;
END \$\$;

-- RDS postgres needs membership in the target role for OWNER assignment
GRANT ${PROJECT}_production TO postgres;

SELECT 'CREATE DATABASE ${PROJECT}_production WITH OWNER ${PROJECT}_production'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${PROJECT}_production')\gexec

\c ${PROJECT}_production
ALTER SCHEMA public OWNER TO ${PROJECT}_production;
EOSQL
echo "✅ Application database ready"

echo ""
echo "📦 Creating Temporal databases and user..."
PGPASSWORD="${MASTER_PASS}" psql -h "${RDS_ENDPOINT}" -U postgres -p 5432 <<EOSQL
DO \$\$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '${PROJECT}_temporal') THEN
    CREATE ROLE ${PROJECT}_temporal WITH LOGIN PASSWORD '${TEMPORAL_PASS}';
  END IF;
END \$\$;

GRANT ${PROJECT}_temporal TO postgres;

SELECT 'CREATE DATABASE ${PROJECT}_temporal WITH OWNER ${PROJECT}_temporal'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${PROJECT}_temporal')\gexec

SELECT 'CREATE DATABASE ${PROJECT}_temporal_visibility WITH OWNER ${PROJECT}_temporal'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${PROJECT}_temporal_visibility')\gexec

\c ${PROJECT}_temporal
ALTER SCHEMA public OWNER TO ${PROJECT}_temporal;

\c ${PROJECT}_temporal_visibility
ALTER SCHEMA public OWNER TO ${PROJECT}_temporal;
EOSQL
echo "✅ Temporal databases ready"

echo ""
echo "════════════════════════════════════════════════════════════════════"
echo "✅ All databases bootstrapped successfully!"
echo "════════════════════════════════════════════════════════════════════"
echo ""
echo "── Application secret (cosmic/focus/production) ──"
echo "   DATABASE_URL: postgresql://${PROJECT}_production:${APP_PASS}@${RDS_ENDPOINT}:5432/${PROJECT}_production?sslmode=require"
echo ""
echo "── Temporal secret (focus/temporal) ──"
echo "   POSTGRES_HOST: ${RDS_ENDPOINT}"
echo "   POSTGRES_USER: ${PROJECT}_temporal"
echo "   POSTGRES_PWD:  ${TEMPORAL_PASS}"
echo "   Databases:     ${PROJECT}_temporal, ${PROJECT}_temporal_visibility"
echo ""
echo "See docs/deploy/setup-checklist.md for the full list of secrets."
