#!/bin/bash
set -e

# Colors for better readability
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HELLO_AGENT_DIR="$PROJECT_ROOT/hello-agent"

# Check if agentctl is in PATH, otherwise use local version
if command -v agentctl &> /dev/null; then
  AGENTCTL="agentctl"
  echo -e "${GREEN}Using system-installed agentctl${NC}"
else
  AGENTCTL="$PROJECT_ROOT/agentctl/agentctl"
  echo -e "${YELLOW}Using local agentctl${NC}"
  
  # Build agentctl if necessary
  echo -e "${BLUE}--- Building agentctl ---${NC}"
  (cd "$PROJECT_ROOT/agentctl" && go build -o agentctl .)
  echo -e "${GREEN}✓ agentctl built successfully${NC}"
fi

echo -e "${BLUE}==== Hello Agent Integration Test with New agentctl Commands ====${NC}"

# Prerequisites
echo -e "${BLUE}Setting up prerequisites...${NC}"
./scripts/setup_prereqs.sh

# Agentctl is already set up in the earlier step

# Build hello-agent using agentctl
echo -e "${BLUE}--- Building Hello Agent Using agentctl ---${NC}"
$AGENTCTL build "$HELLO_AGENT_DIR" --env microk8s
echo -e "${GREEN}✓ Hello Agent built successfully${NC}"

# Deploy hello-agent using agentctl
echo -e "${BLUE}--- Deploying Hello Agent Using agentctl ---${NC}"
$AGENTCTL deploy "$HELLO_AGENT_DIR"
echo -e "${GREEN}✓ Hello Agent deployment initiated${NC}"

# Agent type and namespaces derived from agent.yaml
AGENT_NAME="hello-agent"
AGENT_NAMESPACE="agent-hello" # Based on agent type in hello-agent/agent.yaml
AGENT_POD_LABEL="agent-name=${AGENT_NAME}"

# Wait for pod
echo -e "${BLUE}--- Waiting for Hello Agent Pod to be Created ---${NC}"
sleep 5
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

# Agent status using agentctl
echo -e "${BLUE}--- Agent Status via agentctl ---${NC}"
$AGENTCTL status ${AGENT_NAME}

# Agent logs using agentctl
echo -e "${BLUE}--- Agent Logs (Database Connection) via agentctl ---${NC}"
$AGENTCTL logs ${AGENT_NAME} | grep -E --color=always "PostgreSQL|database|schema|successfully connected|connection|Error"

echo -e "${BLUE}--- Full Agent Logs via agentctl ---${NC}"
$AGENTCTL logs ${AGENT_NAME}

# Operator logs
echo -e "${BLUE}--- Checking Operator Logs ---${NC}"
OPERATOR_POD=$(kubectl get pod -l app.kubernetes.io/component=agent-operator -n agentbox-system -o jsonpath='{.items[0].metadata.name}')
kubectl logs ${OPERATOR_POD} -n agentbox-system | grep -E --color=always "${AGENT_NAME}|credentials|secret|role|Failed" | tail -20

# Test message sending
echo -e "${BLUE}--- Testing Message Sending via agentctl ---${NC}"
$AGENTCTL message ${AGENT_NAME} --payload='{"text": "Hello from test script"}'

# State verification (Postgres)
echo -e "${BLUE}--- Verifying State in Postgres ---${NC}"
echo "You may need to run a psql command or use a script to check agent_state table for agent_id=${AGENT_NAME}."

echo -e "${GREEN}--- Hello Agent Integration Test Complete ---${NC}"
echo -e "${BLUE}--- One-Command Alternative ---${NC}"
echo -e "You can also use the launch command to build, deploy, and tail logs in one step:"
echo -e "${YELLOW}$AGENTCTL launch $HELLO_AGENT_DIR${NC}"
