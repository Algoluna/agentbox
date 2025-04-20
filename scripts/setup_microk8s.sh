#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

echo "--- Checking microk8s Installation and Configuration ---"

# 1. Check if microk8s command exists
if ! command -v microk8s &> /dev/null; then
    echo "Error: 'microk8s' command not found. Please install microk8s."
    exit 1
fi
echo "Microk8s command found."

# Function to run microk8s commands, trying without sudo first if user is in the group
# Stores whether sudo was needed in a global variable
SUDO_USED_FOR_MICROK8S=false
run_microk8s() {
    if groups $USER | grep &>/dev/null '\bmicrok8s\b'; then
        # User is in the group, run without sudo
        SUDO_USED_FOR_MICROK8S=false
        microk8s "$@"
    else
        # User not in group, try with sudo
        SUDO_USED_FOR_MICROK8S=true
        sudo microk8s "$@"
    fi
}

# 2. Check microk8s status and attempt to start if needed
echo "Checking microk8s status..."
if ! run_microk8s status --wait-ready --timeout=10 &> /dev/null; then
    echo "Microk8s not ready, attempting to start..."
    if ! run_microk8s start; then
        echo "Error: Failed to start microk8s. Please check 'microk8s status' manually."
        exit 1
    fi
    echo "Waiting for microk8s to become ready after start..."
    if ! run_microk8s status --wait-ready --timeout=60; then
        echo "Error: microk8s did not become ready after starting. Please check 'microk8s status'."
        exit 1
    fi
    echo "Microk8s started successfully."
else
    echo "Microk8s is running and ready."
fi


# 3. Check if user is in microk8s group and attempt to add if needed
if ! groups $USER | grep &>/dev/null '\bmicrok8s\b'; then
    echo "Warning: User $USER is not in the 'microk8s' group."
    echo "         Attempting to add user to the group for sudo-less operation..."
    if sudo usermod -a -G microk8s $USER; then
        echo "Successfully added user $USER to the 'microk8s' group."
        echo "IMPORTANT: You MUST start a new shell session or log out/log back in for this change to take effect."
        # Set flag so subsequent run_microk8s calls in this script execution still use sudo
        SUDO_USED_FOR_MICROK8S=true
    else
        echo "Error: Failed to add user $USER to the 'microk8s' group. Please do this manually."
        echo "       Command: sudo usermod -a -G microk8s \$USER"
        # Continue, but subsequent commands will require sudo
        SUDO_USED_FOR_MICROK8S=true
    fi
else
    echo "User $USER is already in the 'microk8s' group."
    SUDO_USED_FOR_MICROK8S=false # Ensure flag is false if already in group
fi

# Redefine run_microk8s to use the determined sudo requirement consistently
# This avoids re-checking the group which won't reflect the change in the current session
run_microk8s_setup() {
    if [ "$SUDO_USED_FOR_MICROK8S" = true ]; then
        sudo microk8s "$@"
    else
        microk8s "$@"
    fi
}


# 4. Check required addons and attempt to enable if needed
REQUIRED_ADDONS=("dns" "storage" "registry")
ADDONS_WERE_DISABLED=false

echo "Checking and enabling required addons (${REQUIRED_ADDONS[*]})..."
for addon in "${REQUIRED_ADDONS[@]}"; do
    # Use run_microk8s_setup which respects the sudo decision made earlier
    if run_microk8s_setup status --addon "$addon" | grep -q 'disabled'; then
        echo "Addon '$addon' is disabled. Attempting to enable..."
        if ! run_microk8s_setup enable "$addon"; then
             echo "Error: Failed to enable addon '$addon'. Please try manually: microk8s enable $addon"
             exit 1
        fi
        echo "Addon '$addon' enabled successfully."
        ADDONS_WERE_DISABLED=true
    else
        echo "Addon '$addon' is already enabled."
    fi
done

# If addons were just enabled, wait a bit for them to initialize
if [ "$ADDONS_WERE_DISABLED" = true ]; then
    echo "Waiting for newly enabled addons to initialize..."
    sleep 15
    # Re-check status after enabling addons
    if ! run_microk8s_setup status --wait-ready --timeout=60; then
        echo "Error: microk8s did not become ready after enabling addons. Please check 'microk8s status'."
        exit 1
    fi
fi


echo "--- Microk8s Setup Verification and Configuration Complete ---"
echo "Microk8s is installed, running, and required addons (dns, storage, registry) are enabled."
if [ "$SUDO_USED_FOR_MICROK8S" = true ]; then
     echo "Note: User $USER was added to the 'microk8s' group. Please start a new shell session for sudo-less commands."
fi
