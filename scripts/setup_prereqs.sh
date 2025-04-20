#!/bin/bash
set -e

# Colors for better readability
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values - can be overridden by environment variables
NAMESPACE=${NAMESPACE:-"agentbox-system"}
AGENT_NAMESPACE=${AGENT_NAMESPACE:-"agent-hello-agent"}
SECRET_NAME=${SECRET_NAME:-"agentbox-pg-admin-creds"}
POSTGRES_USER=${POSTGRES_USER:-"postgres"}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-"password"} # CHANGE THIS IN PRODUCTION!
POSTGRES_DB=${POSTGRES_DB:-"agentbox"}
POSTGRES_HOST=${POSTGRES_HOST:-"agentbox-postgresql"}
POSTGRES_PORT=${POSTGRES_PORT:-"5432"}

echo -e "${BLUE}==== AgentBox Prerequisite Setup ====${NC}"
echo -e "${YELLOW}WARNING: For production deployments, please change the default credentials.${NC}"
echo

# Create the main namespace if it doesn't exist
echo -e "${BLUE}Creating namespace: ${NAMESPACE}${NC}"
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Create the agent namespace if it doesn't exist
echo -e "${BLUE}Creating agent namespace: ${AGENT_NAMESPACE}${NC}"
kubectl create namespace ${AGENT_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Create PostgreSQL admin credentials secret
echo -e "${BLUE}Creating PostgreSQL admin credentials secret: ${SECRET_NAME}${NC}"
kubectl create secret generic ${SECRET_NAME} \
  --namespace=${NAMESPACE} \
  --from-literal=POSTGRES_USER=${POSTGRES_USER} \
  --from-literal=POSTGRES_PASSWORD=${POSTGRES_PASSWORD} \
  --from-literal=POSTGRES_DB=${POSTGRES_DB} \
  --from-literal=POSTGRES_HOST=${POSTGRES_HOST} \
  --from-literal=POSTGRES_PORT=${POSTGRES_PORT} \
  --dry-run=client -o yaml | kubectl apply -f -

echo
echo -e "${GREEN}Prerequisites successfully set up!${NC}"
echo -e "${GREEN}You can now deploy AgentBox using Helm with:${NC}"
echo -e "  ${YELLOW}helm upgrade --install agentbox ./helm \\"
echo -e "    --namespace=${NAMESPACE} \\"
echo -e "    -f helm/values.yaml \\"
echo -e "    -f helm/values-microk8s.yaml${NC}"
