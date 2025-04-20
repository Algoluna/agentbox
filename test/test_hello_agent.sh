#!/bin/bash
set -e

# Colors for better readability
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

AGENT_TYPE="hello-agent"
AGENT_NAME="${AGENT_NAME:-$AGENT_TYPE}"
AGENT_NAMESPACE="agent-${AGENT_TYPE}"

echo -e "${BLUE}==== Hello Agent Integration Test ====${NC}"

# Prerequisites
echo -e "${BLUE}Setting up prerequisites...${NC}"
./scripts/setup_prereqs.sh

# Deploy hello-agent
echo -e "${BLUE}--- Deploying Hello Agent Instance ---${NC}"
HELLO_AGENT_IMAGE="localhost:32000/hello-agent:latest"
HELLO_AGENT_IMAGE_ORIGINAL="hello-agent:latest"
sed "s~image: ${HELLO_AGENT_IMAGE_ORIGINAL}~image: ${HELLO_AGENT_IMAGE}~g" hello-agent/config/samples/hello-agent.yaml | \
  awk -v ns="${AGENT_NAMESPACE}" '
    /^metadata:/ { print; inmeta=1; next }
    inmeta && !foundns && /^  name:/ { print; print "  namespace: " ns; foundns=1; next }
    { print }
  ' | kubectl apply -f -
echo -e "${GREEN}Hello Agent Manifest Applied${NC}"

# Wait for pod
echo -e "${BLUE}--- Waiting for Hello Agent Pod to be Created ---${NC}"
sleep 5
AGENT_POD_LABEL="agent-name=${AGENT_NAME}"
for i in {1..12}; do
  echo -e "${YELLOW}Waiting for agent pod ${AGENT_POD_LABEL} (attempt $i/12)...${NC}"
  if kubectl get pods -l ${AGENT_POD_LABEL} -n ${AGENT_NAMESPACE} --no-headers 2>/dev/null | grep -q "."; then
    echo -e "${GREEN}Agent pod found!${NC}"
    break
  fi
  sleep 5
  if [ $i -eq 12 ]; then
    echo -e "${RED}Timed out waiting for agent pod to appear${NC}"
    exit 1
  fi
done

AGENT_POD_NAME=$(kubectl get pod -n ${AGENT_NAMESPACE} -l ${AGENT_POD_LABEL} -o jsonpath='{.items[0].metadata.name}')
echo -e "${GREEN}Found agent pod: ${AGENT_POD_NAME}${NC}"

# Wait for pod to be ready
echo -e "${BLUE}--- Waiting for Pod to Complete ---${NC}"
kubectl wait --for=condition=Ready pod/${AGENT_POD_NAME} -n ${AGENT_NAMESPACE} --timeout=60s || true
for i in {1..12}; do
  STATUS=$(kubectl get pod ${AGENT_POD_NAME} -n ${AGENT_NAMESPACE} -o jsonpath='{.status.phase}')
  echo -e "${YELLOW}Pod status: ${STATUS} (check $i/12)${NC}"
  if [[ "$STATUS" == "Succeeded" || "$STATUS" == "Failed" ]]; then
    break
  fi
  sleep 5
  if [ $i -eq 12 ]; then
    echo -e "${RED}Pod did not complete in expected time${NC}"
  fi
done

# Agent status and logs
echo -e "${BLUE}--- Agent Status ---${NC}"
kubectl get agent ${AGENT_NAME} -n ${AGENT_NAMESPACE}
echo -e "${BLUE}--- Agent Logs (Database Connection) ---${NC}"
kubectl logs ${AGENT_POD_NAME} -n ${AGENT_NAMESPACE} | grep -E --color=always "PostgreSQL|database|schema|successfully connected|connection|Error"
echo -e "${BLUE}--- Full Agent Logs ---${NC}"
kubectl logs ${AGENT_POD_NAME} -n ${AGENT_NAMESPACE}

# Operator logs
echo -e "${BLUE}--- Checking Operator Logs ---${NC}"
OPERATOR_POD=$(kubectl get pod -l app.kubernetes.io/component=agent-operator -n agentbox-system -o jsonpath='{.items[0].metadata.name}')
kubectl logs ${OPERATOR_POD} -n agentbox-system | grep -E --color=always "${AGENT_NAME}|credentials|secret|role|Failed" | tail -20

# CLI checks
echo -e "${BLUE}--- Building and Testing agentctl CLI ---${NC}"
cd "$(dirname "$0")/../agentctl"
go build -o agentctl .
./agentctl status
./agentctl status ${AGENT_NAME}
cd -

# State verification (Postgres)
echo -e "${BLUE}--- Verifying State in Postgres ---${NC}"
echo "You may need to run a psql command or use a script to check agent_state table for agent_id=${AGENT_NAME}."

echo -e "${GREEN}--- Hello Agent Integration Test Complete ---${NC}"
