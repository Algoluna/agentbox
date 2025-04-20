# agent_sdk.types.py

"""
Shared types for agent SDK.
"""

from typing import Any, Optional

class IncomingMessage:
    def __init__(self, id: str, sender: str, reply_to: Optional[str], payload: dict, type: str):
        self.id = id
        self.sender = sender
        self.reply_to = reply_to
        self.payload = payload
        self.type = type

    def __repr__(self):
        return f"IncomingMessage(id={self.id}, sender={self.sender}, type={self.type})"
