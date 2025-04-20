#!/bin/bash
# Test script for demonstrating the simplified agentctl workflow

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
AGENTCTL="$PROJECT_ROOT/agentctl/agentctl"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building agentctl...${NC}"
(cd "$PROJECT_ROOT/agentctl" && go build -o agentctl .)
echo -e "${GREEN}✓ agentctl built successfully${NC}"

# Test the chatbot agent
echo -e "\n${YELLOW}== Testing Chatbot Agent ==${NC}"

echo -e "\n${YELLOW}Building chatbot-agent...${NC}"
$AGENTCTL build "$PROJECT_ROOT/examples/chatbot-agent"
echo -e "${GREEN}✓ chatbot-agent built successfully${NC}"

echo -e "\n${YELLOW}Deploying chatbot-agent...${NC}"
$AGENTCTL deploy "$PROJECT_ROOT/examples/chatbot-agent"
echo -e "${GREEN}✓ chatbot-agent deployed successfully${NC}"

echo -e "\n${YELLOW}Getting chatbot-agent status...${NC}"
$AGENTCTL status chatbot-agent

echo -e "\n${YELLOW}Sending message to chatbot-agent...${NC}"
$AGENTCTL message chatbot-agent --payload='{"text": "Hello from test script"}'

# Test the chatbot router (includes RBAC)
echo -e "\n\n${YELLOW}== Testing Chatbot Router ==${NC}"

echo -e "\n${YELLOW}Building chatbot-router...${NC}"
$AGENTCTL build "$PROJECT_ROOT/examples/chatbot-router"
echo -e "${GREEN}✓ chatbot-router built successfully${NC}"

echo -e "\n${YELLOW}Deploying chatbot-router (with RBAC)...${NC}"
$AGENTCTL deploy "$PROJECT_ROOT/examples/chatbot-router"
echo -e "${GREEN}✓ chatbot-router deployed with RBAC successfully${NC}"

echo -e "\n${YELLOW}Getting chatbot-router status...${NC}"
$AGENTCTL status chatbot-router

echo -e "\n${YELLOW}Sending message to chatbot-router...${NC}"
$AGENTCTL message chatbot-router --payload='{"user_id": "test-user", "text": "Hello from test script"}'

# Show all agents
echo -e "\n\n${YELLOW}== Showing All Agents ==${NC}"
$AGENTCTL status

echo -e "\n${GREEN}All tests completed successfully!${NC}"
