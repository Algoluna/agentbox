# agent_sdk.runtime.context.py

"""
RuntimeContext: Infrastructure integration for ADK-compatible agents.

- Handles persistent state (Postgres)
- Messaging (Valkey/Redis Streams)
- LLM/Embedding stubs
- Status reporting (Postgres)
- Designed for injection into ADK agents (no subclassing required)
"""

from typing import Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from agent_sdk.runtime.messaging import Messaging
    from agent_sdk.runtime.llm import LLMManager
    from agent_sdk.runtime.embedding import EmbeddingManager
    from agent_sdk.types import IncomingMessage

import os
from agent_sdk.runtime.messaging import Messaging
from agent_sdk.runtime.llm import LLMManager
from agent_sdk.runtime.embedding import EmbeddingManager
from agent_sdk.db.state import StateManager
import os

class RuntimeContext:
    def __init__(self, agent_id: str, db_url: str, redis_url: str, agent_type: str = None):
        self.agent_id = agent_id
        self.db_url = db_url
        self.redis_url = redis_url
        self.agent_type = agent_type or os.environ.get("AGENT_TYPE", "hello-agent")
        self._messaging = Messaging(redis_url, self.agent_type, agent_id)
        self._state = StateManager(db_url)
        self._llm = LLMManager()
        self._embedding = EmbeddingManager()

    @classmethod
    def from_env(cls, agent_id: str) -> "RuntimeContext":
        db_url = os.environ.get("DB_URL")
        redis_url = os.environ.get("REDIS_URL")
        agent_type = os.environ.get("AGENT_TYPE", "hello-agent")
        if not db_url:
            raise RuntimeError("DB_URL environment variable is required for agent state persistence.")
        if not redis_url:
            raise RuntimeError("REDIS_URL environment variable is required for messaging.")
        return cls(agent_id, db_url, redis_url, agent_type)

    def receive(self) -> "IncomingMessage":
        # TODO: Block on Redis stream for incoming message
        return self._messaging.receive()

    def load_state(self, agent) -> None:
        # Hydrate agent.state from Postgres
        state = self._state.load_state(self.agent_id)
        if state is not None:
            agent.state = state

    def save_state(self, agent) -> None:
        # Persist agent.state to Postgres
        self._state.save_state(self.agent_id, agent.state)

    def messaging(self) -> "Messaging":
        return self._messaging

    def llm(self) -> "LLMManager":
        return self._llm

    def embedding(self) -> "EmbeddingManager":
        return self._embedding

    def report_status(self, phase: str, step: Optional[str] = None, progress: Optional[str] = None, message: Optional[str] = None):
        # Report agent status to Postgres (agent_status table)
        import psycopg2
        try:
            conn = psycopg2.connect(self.db_url)
            with conn.cursor() as cur:
                cur.execute(
                    """
                    INSERT INTO agent_status (agent_id, phase, step, progress, message, updated_at)
                    VALUES (%s, %s, %s, %s, %s, NOW())
                    ON CONFLICT (agent_id)
                    DO UPDATE SET phase = EXCLUDED.phase, step = EXCLUDED.step, progress = EXCLUDED.progress, message = EXCLUDED.message, updated_at = NOW()
                    """,
                    (self.agent_id, phase, step, progress, message)
                )
                conn.commit()
            conn.close()
        except Exception as e:
            import logging
            logging.getLogger("RuntimeContext").error(f"Error reporting status: {e}")

    def mark_completed(self, message: Optional[str] = None):
        # Mark agent as completed in status table
        self.report_status(phase="completed", message=message)

    def mark_failed(self, message: Optional[str] = None):
        # Mark agent as failed in status table
        self.report_status(phase="failed", message=message)
