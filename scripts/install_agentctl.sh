#!/bin/bash
# Script to build agentctl and install it to user's local bin directory

set -e

# Colors for better readability
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
AGENTCTL_DIR="$PROJECT_ROOT/agentctl"

# User bin directories (in order of preference)
USER_BIN_DIRS=(
  "$HOME/.local/bin"
  "$HOME/bin"
)

# Find the first existing bin directory or create ~/.local/bin
USER_BIN=""
for dir in "${USER_BIN_DIRS[@]}"; do
  if [[ -d "$dir" && "$PATH" == *"$dir"* ]]; then
    USER_BIN="$dir"
    break
  fi
done

# If no bin directory in PATH, create ~/.local/bin
if [ -z "$USER_BIN" ]; then
  USER_BIN="$HOME/.local/bin"
  mkdir -p "$USER_BIN"
  echo -e "${YELLOW}Created $USER_BIN directory${NC}"
  
  # Check if this directory is in PATH
  if [[ "$PATH" != *"$USER_BIN"* ]]; then
    echo -e "${YELLOW}Adding $USER_BIN to your PATH${NC}"
    echo -e "\n# Add ~/.local/bin to PATH for agentctl" >> "$HOME/.bashrc"
    echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$HOME/.bashrc"
    echo -e "\n${YELLOW}Please run: source ~/.bashrc${NC}"
    echo -e "${YELLOW}Or open a new terminal for the changes to take effect${NC}"
  fi
fi

echo -e "${BLUE}Building agentctl...${NC}"
cd "$AGENTCTL_DIR"
go build -o agentctl .

echo -e "${BLUE}Installing agentctl to $USER_BIN${NC}"
cp agentctl "$USER_BIN/"
chmod +x "$USER_BIN/agentctl"

echo -e "${GREEN}✓ agentctl installed successfully to $USER_BIN/agentctl${NC}"
echo -e "Try it with: agentctl status"

# Check if directory is in current PATH
if [[ "$PATH" != *"$USER_BIN"* ]]; then
  echo -e "${YELLOW}Note: $USER_BIN is not in your current PATH.${NC}"
  echo -e "${YELLOW}Changes have been made to ~/.bashrc, but you'll need to:${NC}"
  echo -e "${YELLOW}  source ~/.bashrc${NC}"
  echo -e "${YELLOW}Or open a new terminal for the changes to take effect.${NC}"
fi
