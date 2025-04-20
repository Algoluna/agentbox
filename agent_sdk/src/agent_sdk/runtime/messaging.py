# agent_sdk.runtime.messaging.py

"""
Messaging abstraction for agent SDK.
Handles Redis/Valkey Streams for message receipt and delivery.
"""

import redis
import json
import logging
import time
from agent_sdk.types import IncomingMessage

class Messaging:
    def __init__(self, redis_url: str, agent_type: str, agent_id: str):
        self.redis_url = redis_url
        self.agent_type = agent_type
        self.agent_id = agent_id
        self.stream_key = f"agent:{agent_type}:{agent_id}:inbox"
        self.logger = logging.getLogger("Messaging")
        # Use username from env or default to agent_helloagent
        import os
        username = os.environ.get("REDIS_USERNAME", f"agent_{agent_id.replace('-', '').replace('_', '')}")
        self.logger.info(f"Connecting to Redis with username={username}")
        from urllib.parse import urlparse
        url = urlparse(redis_url)
        password = url.password
        host = url.hostname
        port = url.port or 6379
        self.redis = redis.Redis(
            host=host,
            port=port,
            username=username,
            password=password,
            decode_responses=True,
            socket_connect_timeout=5
        )
        self.last_id = "0-0"

    def receive(self):
        # Block on Redis stream for incoming message
        while True:
            try:
                streams = self.redis.xread({self.stream_key: self.last_id}, block=0, count=1)
                if streams:
                    _, messages = streams[0]
                    for msg_id, fields in messages:
                        self.last_id = msg_id
                        try:
                            data = json.loads(fields.get("data", "{}"))
                            return IncomingMessage(
                                id=data.get("id", msg_id),
                                sender=data.get("sender", ""),
                                reply_to=data.get("reply_to"),
                                payload=data.get("payload", {}),
                                type=data.get("type", "")
                            )
                        except Exception as e:
                            self.logger.error(f"Error parsing message: {e}")
            except Exception as e:
                self.logger.error(f"Error reading from Redis stream: {e}")
                time.sleep(1)

    def reply(self, message, payload):
        # Send reply to Redis stream (reply_to or sender's inbox)
        target_stream = message.reply_to or f"agent:{message.sender}:inbox"
        try:
            msg = {
                "id": message.id,
                "sender": self.agent_id,
                "reply_to": message.id,
                "payload": json.dumps(payload),
                "type": "reply"
            }
            self.redis.xadd(target_stream, {"data": json.dumps(msg)})
        except Exception as e:
            self.logger.error(f"Error sending reply: {e}")
