"""
ControlPlaneAgent: Base class for agents that interact with the Kubernetes control plane.
Provides a pre-configured AgentManager and methods for agent lifecycle operations, including TTL refresh.
"""

import os
import logging
import time
from typing import Dict, Any, Optional
from abc import ABC, abstractmethod

from google.adk.agents import Agent
from agent_sdk.runtime.context import RuntimeContext
from agent_sdk.control.agent_manager import AgentManager

class ControlPlaneAgent(ABC):
    """
    Base class for agents that need to interact with the Kubernetes control plane.
    """

    def __init__(self, agent_id: str, agent_type: str, initial_state: Dict[str, Any] = None,
                 namespace: str = None, service_account: str = None):
        self.agent_id = agent_id
        self.agent_type = agent_type
        self.logger = logging.getLogger(f"{self.agent_type}-{self.agent_id}")

        self.namespace = namespace or os.environ.get("NAMESPACE", "default")
        self.service_account = service_account or os.environ.get("SERVICE_ACCOUNT_NAME")

        self.context = RuntimeContext.from_env(self.agent_id)
        self.agent = self._create_agent(initial_state or {})

        self.agent_manager = AgentManager(namespace=self.namespace)
        self.logger.info(f"ControlPlaneAgent initialized: {self.agent_id} ({self.agent_type})")

    def _create_agent(self, initial_state: Dict[str, Any]) -> Agent:
        return Agent(
            name=self.agent_id,
            description=f"Control plane agent: {self.agent_type}",
            state=initial_state,
            tools=[]
        )

    def run(self):
        try:
            self.context.load_state(self.agent)
            self.logger.info(f"Agent state loaded for {self.agent_id}")
            self.initialize()
            self.logger.info(f"Starting message processing loop for {self.agent_id}")
            self._process_messages()
        except KeyboardInterrupt:
            self.logger.info(f"Shutting down {self.agent_id}")
        except Exception as e:
            self.logger.error(f"Error in {self.agent_id}: {e}", exc_info=True)
            self.context.mark_failed(str(e))
        finally:
            self.cleanup()
            self.context.save_state(self.agent)

    def _process_messages(self):
        self.logger.info(f"Agent {self.agent_id} waiting for messages")
        last_periodic_time = time.time()
        periodic_interval = self.get_periodic_interval()
        while True:
            if periodic_interval > 0:
                now = time.time()
                if now - last_periodic_time > periodic_interval:
                    self.periodic_operation()
                    last_periodic_time = now
                    self.context.save_state(self.agent)
            message = self.context.receive()
            self.logger.info(f"Received message from {message.sender}: {message.payload}")
            try:
                self.handle_message(message)
                self.context.save_state(self.agent)
            except Exception as e:
                self.logger.error(f"Error processing message: {e}", exc_info=True)
                try:
                    self.context.messaging().reply(message, {
                        "text": f"Error processing message: {str(e)}"
                    })
                except Exception:
                    self.logger.error("Failed to send error response", exc_info=True)

    def provision_agent(self,
                      name: str,
                      agent_type: str,
                      image: str,
                      run_once: bool = False,
                      env_vars: Optional[Dict[str, str]] = None,
                      service_account_name: Optional[str] = None,
                      ttl: Optional[int] = None) -> bool:
        self.logger.info(f"Provisioning agent: {name} (type: {agent_type})")
        return self.agent_manager.create_agent(
            name=name,
            agent_type=agent_type,
            image=image,
            run_once=run_once,
            env_vars=env_vars,
            service_account_name=service_account_name,
            ttl=ttl
        )

    def deprovision_agent(self, name: str) -> bool:
        self.logger.info(f"Deprovisioning agent: {name}")
        return self.agent_manager.delete_agent(name)

    def wait_for_agent_ready(self, name: str, timeout_seconds: int = 60) -> bool:
        return self.agent_manager.wait_for_agent_ready(name, timeout_seconds)

    def refresh_agent_ttl(self, name: str) -> bool:
        return self.agent_manager.update_last_activity(name)

    @abstractmethod
    def initialize(self):
        pass

    @abstractmethod
    def handle_message(self, message):
        pass

    @abstractmethod
    def cleanup(self):
        pass

    def periodic_operation(self):
        pass

    def get_periodic_interval(self) -> int:
        return 0
