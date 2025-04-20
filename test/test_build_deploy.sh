#!/bin/bash
# Test script for demonstrating the agentctl build and deploy commands

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Check if agentctl is in PATH, otherwise use local version
if command -v agentctl &> /dev/null; then
  AGENTCTL="agentctl"
  echo -e "${GREEN}Using system-installed agentctl${NC}"
else
  AGENTCTL="$PROJECT_ROOT/agentctl/agentctl"
  echo -e "${YELLOW}Using local agentctl${NC}"
  
  # Build agentctl if necessary
  echo -e "${YELLOW}Building agentctl...${NC}"
  (cd "$PROJECT_ROOT/agentctl" && go build -o agentctl .)
  echo -e "${GREEN}✓ agentctl built successfully${NC}"
fi

# agentctl is already set up above

# Path to the chatbot agent example
AGENT_DIR="$PROJECT_ROOT/examples/chatbot-agent"

echo -e "\n${YELLOW}==== Testing agentctl build ====${NC}"
echo -e "Building agent in $AGENT_DIR"
$AGENTCTL build "$AGENT_DIR"
echo -e "${GREEN}✓ Build command completed successfully${NC}"

echo -e "\n${YELLOW}==== Testing agentctl deploy ====${NC}"
echo -e "Deploying agent from $AGENT_DIR"
$AGENTCTL deploy "$AGENT_DIR"
echo -e "${GREEN}✓ Deploy command completed successfully${NC}"

echo -e "\n${YELLOW}==== Testing agentctl status ====${NC}"
$AGENTCTL status chatbot-agent

echo -e "\n${YELLOW}==== Testing agentctl launch ====${NC}"
echo -e "This combines build and deploy into a single command"
echo -e "To use: $AGENTCTL launch $AGENT_DIR"
echo -e "${GREEN}✓ All tests completed successfully!${NC}"

cat << EOF

${YELLOW}===========================================${NC}
${GREEN}USAGE SUMMARY${NC}
${YELLOW}===========================================${NC}

1. ${GREEN}Build an agent:${NC}
   $AGENTCTL build [directory]

2. ${GREEN}Deploy an agent:${NC}
   $AGENTCTL deploy [directory]  

3. ${GREEN}One-command workflow:${NC}
   $AGENTCTL launch [directory]

4. ${GREEN}Monitor agents:${NC}
   $AGENTCTL status [agent-name]
   $AGENTCTL logs <agent-name> [--follow]

5. ${GREEN}Interact with agents:${NC}
   $AGENTCTL message <agent-name> --payload='{"key":"value"}'

${YELLOW}===========================================${NC}
EOF
