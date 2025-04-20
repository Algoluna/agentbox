"""
AgentManager: Framework class for agent control plane operations.
Provides abstraction for creating, deleting, and monitoring agents, including TTL and LastActivityTime support.
"""

import logging
import os
import time
from typing import Dict, Any, List, Optional
from kubernetes import client, config
from kubernetes.client.rest import ApiException
from kubernetes.client import CustomObjectsApi
from datetime import datetime, timezone

class AgentManager:
    """
    Manages agent lifecycle operations using the Kubernetes API.
    """

    def __init__(self, namespace: str = "default"):
        """
        Initialize the AgentManager with Kubernetes client.

        Args:
            namespace: Kubernetes namespace for agent operations
        """
        self.logger = logging.getLogger("AgentManager")
        self.namespace = namespace

        # Initialize Kubernetes client
        try:
            config.load_incluster_config()
        except config.ConfigException:
            config.load_kube_config()

        self.custom_api = CustomObjectsApi()
        self.group = "agents.algoluna.com"
        self.version = "v1alpha1"
        self.plural = "agents"

        self.logger.info(f"AgentManager initialized for namespace {namespace}")

    def create_agent(self,
                    name: str,
                    agent_type: str,
                    image: str,
                    run_once: bool = False,
                    env_vars: Optional[Dict[str, str]] = None,
                    service_account_name: Optional[str] = None,
                    ttl: Optional[int] = None) -> bool:
        """
        Create a new agent custom resource.

        Args:
            name: Name of the agent
            agent_type: Type of the agent
            image: Container image for the agent
            run_once: Whether the agent should run once or continuously
            env_vars: Environment variables for the agent
            service_account_name: Optional service account to use
            ttl: Time to live in seconds, 0 means no TTL (default: None)

        Returns:
            True if successful, False otherwise
        """
        env = []
        if env_vars:
            for key, value in env_vars.items():
                env.append({"name": key, "value": value})

        agent_body = {
            "apiVersion": f"{self.group}/{self.version}",
            "kind": "Agent",
            "metadata": {
                "name": name,
                "namespace": self.namespace
            },
            "spec": {
                "type": agent_type,
                "image": image,
                "runOnce": run_once,
                "env": env
            }
        }

        if service_account_name:
            agent_body["spec"]["serviceAccountName"] = service_account_name
        if ttl is not None and ttl > 0:
            agent_body["spec"]["ttl"] = ttl

        try:
            self.custom_api.create_namespaced_custom_object(
                group=self.group,
                version=self.version,
                namespace=self.namespace,
                plural=self.plural,
                body=agent_body
            )
            self.logger.info(f"Agent {name} created successfully")
            return True
        except ApiException as e:
            self.logger.error(f"Failed to create agent {name}: {e}")
            return False

    def delete_agent(self, name: str) -> bool:
        """
        Delete an agent custom resource.

        Args:
            name: Name of the agent to delete

        Returns:
            True if successful, False otherwise
        """
        try:
            self.custom_api.delete_namespaced_custom_object(
                group=self.group,
                version=self.version,
                namespace=self.namespace,
                plural=self.plural,
                name=name
            )
            self.logger.info(f"Agent {name} deleted successfully")
            return True
        except ApiException as e:
            self.logger.error(f"Failed to delete agent {name}: {e}")
            return False

    def get_agent_status(self, name: str) -> Optional[Dict[str, Any]]:
        """
        Get the status of an agent.

        Args:
            name: Name of the agent

        Returns:
            Agent status dictionary or None if agent not found
        """
        try:
            agent = self.custom_api.get_namespaced_custom_object(
                group=self.group,
                version=self.version,
                namespace=self.namespace,
                plural=self.plural,
                name=name
            )
            return agent.get("status", {})
        except ApiException as e:
            self.logger.error(f"Failed to get status for agent {name}: {e}")
            return None

    def list_agents(self, label_selector: str = None) -> List[Dict[str, Any]]:
        """
        List all agents, optionally filtered by labels.

        Args:
            label_selector: Label selector to filter agents

        Returns:
            List of agent resources
        """
        try:
            agents = self.custom_api.list_namespaced_custom_object(
                group=self.group,
                version=self.version,
                namespace=self.namespace,
                plural=self.plural,
                label_selector=label_selector
            )
            return agents.get("items", [])
        except ApiException as e:
            self.logger.error(f"Failed to list agents: {e}")
            return []

    def is_agent_ready(self, name: str) -> bool:
        """
        Check if an agent is in Running phase.

        Args:
            name: Name of the agent

        Returns:
            True if agent is ready, False otherwise
        """
        status = self.get_agent_status(name)
        if not status:
            return False
        return status.get("phase") == "Running"

    def wait_for_agent_ready(self, name: str, timeout_seconds: int = 60) -> bool:
        """
        Wait for an agent to reach Running phase.

        Args:
            name: Name of the agent
            timeout_seconds: Maximum time to wait in seconds

        Returns:
            True if agent reached Running phase, False if timed out
        """
        start_time = time.time()
        while time.time() - start_time < timeout_seconds:
            if self.is_agent_ready(name):
                return True
            self.logger.info(f"Waiting for agent {name} to be ready...")
            time.sleep(2)
        self.logger.warning(f"Timed out waiting for agent {name} to be ready")
        return False

    def update_last_activity(self, name: str) -> bool:
        """
        Update the last activity timestamp for an agent to prevent TTL expiration.

        Args:
            name: Name of the agent

        Returns:
            True if successful, False otherwise
        """
        try:
            agent = self.custom_api.get_namespaced_custom_object(
                group=self.group,
                version=self.version,
                namespace=self.namespace,
                plural=self.plural,
                name=name
            )
            now = datetime.now(timezone.utc).isoformat()
            if "spec" not in agent:
                agent["spec"] = {}
            agent["spec"]["lastActivityTime"] = now
            self.custom_api.patch_namespaced_custom_object(
                group=self.group,
                version=self.version,
                namespace=self.namespace,
                plural=self.plural,
                name=name,
                body={"spec": {"lastActivityTime": now}}
            )
            self.logger.info(f"Updated last activity time for agent {name}")
            return True
        except ApiException as e:
            self.logger.error(f"Failed to update last activity for agent {name}: {e}")
            return False
