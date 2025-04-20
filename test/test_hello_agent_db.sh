#!/bin/bash
# DEPRECATED: All hello-agent integration and database checks are now in test/test_hello_agent.sh.
# This script is retained for reference only and should not be used.

set -e

# Colors for better readability
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

AGENT_TYPE="hello-agent"
# AGENT_NAME is optional; default to AGENT_TYPE if not set
AGENT_NAME="${AGENT_NAME:-$AGENT_TYPE}"
AGENT_NAMESPACE="agent-${AGENT_TYPE}"

echo -e "${BLUE}==== Testing Hello Agent Database Connection ====${NC}"

# Make sure prerequisites are set up
echo -e "${BLUE}Setting up prerequisites if needed...${NC}"
./scripts/setup_prereqs.sh

echo -e "${BLUE}--- Deploying Hello Agent Instance ---${NC}"
# Use the hello-agent.yaml from the samples directory
HELLO_AGENT_IMAGE="localhost:32000/hello-agent:latest"
HELLO_AGENT_IMAGE_ORIGINAL="hello-agent:latest"

echo -e "${BLUE}Replacing image and setting namespace in hello-agent/config/samples/hello-agent.yaml with ${HELLO_AGENT_IMAGE} and namespace ${AGENT_NAMESPACE}${NC}"
sed "s~image: ${HELLO_AGENT_IMAGE_ORIGINAL}~image: ${HELLO_AGENT_IMAGE}~g" hello-agent/config/samples/hello-agent.yaml | \
  awk -v ns="${AGENT_NAMESPACE}" '
    /^metadata:/ { print; inmeta=1; next }
    inmeta && !foundns && /^  name:/ { print; print "  namespace: " ns; foundns=1; next }
    { print }
  ' | kubectl apply -f -
echo -e "${GREEN}Hello Agent Manifest Applied${NC}"

echo -e "${BLUE}--- Waiting for Hello Agent Pod to be Created ---${NC}"
# Wait for the agent pod, which should be created by the operator
sleep 5 # Give operator time to react and create the pod
AGENT_POD_LABEL="agent-name=${AGENT_NAME}"

# Check if agent pod exists
echo -e "${BLUE}Checking for agent pod...${NC}"
kubectl get pods -l ${AGENT_POD_LABEL} -n ${AGENT_NAMESPACE} --no-headers || echo "No agent pod found yet, waiting..."

# Wait for pod to appear
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

echo -e "${BLUE}--- Waiting for Pod to Complete ---${NC}"
# Wait for the pod to reach Succeeded state
kubectl wait --for=condition=Ready pod/${AGENT_POD_NAME} -n ${AGENT_NAMESPACE} --timeout=60s || true

# Keep checking until the pod reaches Succeeded or Failed
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

echo -e "${BLUE}--- Agent Status ---${NC}"
kubectl get agent ${AGENT_NAME} -n ${AGENT_NAMESPACE}

echo -e "${BLUE}--- Agent Logs (Checking Database Connection) ---${NC}"
# Get agent logs and colorize database-related messages
kubectl logs ${AGENT_POD_NAME} -n ${AGENT_NAMESPACE} | grep -E --color=always "PostgreSQL|database|schema|successfully connected|connection|Error"

echo -e "${BLUE}--- Full Agent Logs ---${NC}"
kubectl logs ${AGENT_POD_NAME} -n ${AGENT_NAMESPACE}

echo -e "${BLUE}--- Checking Operator Logs ---${NC}"
OPERATOR_POD=$(kubectl get pod -l app.kubernetes.io/component=agent-operator -n agentbox-system -o jsonpath='{.items[0].metadata.name}')
kubectl logs ${OPERATOR_POD} -n agentbox-system | grep -E --color=always "${AGENT_NAME}|credentials|secret|role|Failed" | tail -20

echo -e "${GREEN}--- Testing Complete ---${NC}"
