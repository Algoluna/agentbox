"""
ChatbotRouter: Routes messages to per-user chatbot instances.
Built on the ControlPlaneAgent base class, supports ephemeral agents with TTL.
"""

import logging
import os
import json
import time
import uuid
from typing import Dict, Any

from agent_sdk.control import ControlPlaneAgent
from agent_sdk.types import IncomingMessage

# Configure logging
logging.basicConfig(level=logging.INFO)

# Initial state for the router agent
INITIAL_STATE = {
    "user_agents": {},  # Maps user IDs to agent names
    "agent_status": {},  # Tracks agent status (running, stopped)
    "agent_last_active": {},  # Tracks when agents were last used
}

class ChatbotRouter(ControlPlaneAgent):
    """
    Router agent for chatbot instances.
    """

    def __init__(self):
        agent_id = os.environ.get("AGENT_ID", "chatbot-router")
        super().__init__(
            agent_id=agent_id,
            agent_type="chatbot-router",
            initial_state=INITIAL_STATE,
        )

    def initialize(self):
        self.logger.info("ChatbotRouter initialized and ready for messages")

    def get_periodic_interval(self) -> int:
        # No manual cleanup needed; TTL is enforced by the control plane
        return 0

    def get_agent_for_user(self, user_id: str) -> str:
        """Get or create agent for user."""
        if user_id in self.agent.state["user_agents"]:
            agent_name = self.agent.state["user_agents"][user_id]
            self.logger.info(f"Found existing agent {agent_name} for user {user_id}")
            return agent_name

        # Create new agent for user
        agent_name = f"chatbot-user-{user_id}-{str(uuid.uuid4())[:8]}"
        self.logger.info(f"Creating new agent {agent_name} for user {user_id}")

        env_vars = {
            "AGENT_TYPE": "chatbot-agent",
            "AGENT_ID": agent_name,
            "MODEL_NAME": "models/gemini-flash-2.0"
        }

        # Set a TTL of 30 minutes (1800 seconds)
        ttl = 1800

        success = self.provision_agent(
            name=agent_name,
            agent_type="chatbot-agent",
            image="chatbot-agent:latest",
            run_once=False,
            env_vars=env_vars,
            ttl=ttl
        )

        if success:
            self.agent.state["user_agents"][user_id] = agent_name
            self.agent.state["agent_status"][agent_name] = "provisioning"
            self.agent.state["agent_last_active"][agent_name] = time.time()
            self.logger.info(f"Agent {agent_name} created for user {user_id}")
        else:
            raise RuntimeError(f"Failed to create agent for user {user_id}")

        return agent_name

    def handle_message(self, message: IncomingMessage):
        """Process an incoming message."""
        user_id = message.sender
        if not message.payload or "text" not in message.payload:
            self.logger.warning("Received message with no text payload")
            self.context.messaging().reply(message, {
                "text": "Sorry, I couldn't understand your message."
            })
            return

        user_message = message.payload["text"]

        # Route message to user-specific agent
        success = self.route_message(user_id, user_message, message.id)

        if not success:
            self.context.messaging().reply(message, {
                "text": "Sorry, I'm having trouble processing your message right now."
            })

    def route_message(self, user_id: str, message_text: str, message_id: str) -> bool:
        """Route a message to the appropriate chatbot agent."""
        agent_name = self.get_agent_for_user(user_id)

        # Update last active timestamp in our state and refresh TTL in the CR
        self.agent.state["agent_last_active"][agent_name] = time.time()
        self.refresh_agent_ttl(agent_name)

        # Check if agent is ready
        if not self.wait_for_agent_ready(agent_name, timeout_seconds=60):
            self.logger.error(f"Agent {agent_name} failed to become ready")
            return False

        # Forward the message to the user's agent
        try:
            target_stream = f"agent:{agent_name}:inbox"
            msg_payload = {
                "id": message_id,
                "sender": user_id,
                "reply_to": f"agent:chatbot-router:inbox",
                "payload": {"text": message_text},
                "type": "message"
            }
            redis_client = self.context.messaging()._redis
            redis_client.xadd(target_stream, {"data": json.dumps(msg_payload)})
            self.logger.info(f"Message routed to agent {agent_name}")
            return True
        except Exception as e:
            self.logger.error(f"Failed to route message to agent {agent_name}: {e}")
            return False

    def cleanup(self):
        self.logger.info("Cleaning up before shutdown")

def main():
    router = ChatbotRouter()
    router.run()

if __name__ == "__main__":
    main()
