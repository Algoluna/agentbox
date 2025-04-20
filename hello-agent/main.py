from agent_sdk.runtime.registry import register_agent
from agent_sdk.types import IncomingMessage
import logging
import time

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('hello-agent')

@register_agent("hello-agent")
class HelloAgent:
    def __init__(self, name):
        self.name = name
        self.state = {}

    def on_message(self, message: IncomingMessage):
        logger.info(f"Received message: {message}")
        self.state["last_message"] = message.payload
        # Simulate work
        for i in range(5):
            logger.info(f"Hello... iteration {i}")
            time.sleep(2)
        # Example reply (stub)
        self.ctx.messaging().reply(message, {"response": "Hello from agent!"})
