# agent_sdk.db.state.py

"""
State persistence for agent SDK.
Handles loading and saving agent state to Postgres.
"""

import psycopg2
import json
import logging

class StateManager:
    def __init__(self, db_url: str):
        self.db_url = db_url
        self.logger = logging.getLogger("StateManager")

    def load_state(self, agent_id: str):
        try:
            conn = psycopg2.connect(self.db_url)
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT state_json FROM agent_state WHERE agent_id = %s",
                    (agent_id,)
                )
                row = cur.fetchone()
                if row and row[0]:
                    return json.loads(row[0])
            conn.close()
        except Exception as e:
            self.logger.error(f"Error loading state for agent {agent_id}: {e}")
        return None

    def save_state(self, agent_id: str, state: dict):
        try:
            conn = psycopg2.connect(self.db_url)
            with conn.cursor() as cur:
                cur.execute(
                    """
                    INSERT INTO agent_state (agent_id, state_json, updated_at)
                    VALUES (%s, %s, NOW())
                    ON CONFLICT (agent_id)
                    DO UPDATE SET state_json = EXCLUDED.state_json, updated_at = NOW()
                    """,
                    (agent_id, json.dumps(state))
                )
                conn.commit()
            conn.close()
        except Exception as e:
            self.logger.error(f"Error saving state for agent {agent_id}: {e}")
