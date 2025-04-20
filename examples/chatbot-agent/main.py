"""
Chatbot Agent: A simple per-user chatbot that uses a model to respond to user messages.
Maintains conversation history in persistent state.
"""

import logging
import os
from typing import Dict, Any, List
from google.adk.agents import Agent
from agent_sdk.runtime.registry import register_agent
from agent_sdk.runtime.context import RuntimeContext
from agent_sdk.runtime.model import ModelManager

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("chatbot-agent")

# Initial state for conversation history
INITIAL_STATE = {
    "conversations": {},  # Stores conversations by user ID
}

def get_conversation_history(state: Dict[str, Any], user_id: str) -> List[Dict[str, str]]:
    if user_id not in state["conversations"]:
        state["conversations"][user_id] = []
    return state["conversations"][user_id]

def add_to_conversation(state: Dict[str, Any], user_id: str, role: str, content: str):
    history = get_conversation_history(state, user_id)
    history.append({"role": role, "content": content})
    if len(history) > 10:
        history.pop(0)

def create_prompt(conversation: List[Dict[str, str]], new_message: str) -> str:
    prompt = "You are a helpful, friendly chatbot. Respond to the following conversation and message:\n\n"
    for message in conversation:
        if message["role"] == "user":
            prompt += f"User: {message['content']}\n"
        else:
            prompt += f"Assistant: {message['content']}\n"
    prompt += f"User: {new_message}\nAssistant:"
    return prompt

@register_agent("chatbot-agent")
def create_agent() -> Agent:
    agent = Agent(
        name="chatbot-agent",
        description="A simple chatbot that uses a model to respond to user messages.",
        state=INITIAL_STATE,
        tools=[],
    )
    return agent

def main():
    agent_id = os.environ.get("AGENT_ID", "chatbot-agent")
    context = RuntimeContext.from_env(agent_id)
    agent = create_agent()
    context.load_state(agent)
    logger.info(f"Chatbot agent {agent_id} started and waiting for messages")

    model = ModelManager()

    try:
        while True:
            message = context.receive()
            logger.info(f"Received message from {message.sender}: {message.payload}")

            if not message.payload or "text" not in message.payload:
                logger.warning("Received message with no text payload")
                continue

            user_message = message.payload["text"]
            user_id = message.sender

            add_to_conversation(agent.state, user_id, "user", user_message)
            conversation = get_conversation_history(agent.state, user_id)
            prompt = create_prompt(conversation, user_message)

            model_response = model.ask(prompt)

            add_to_conversation(agent.state, user_id, "assistant", model_response)
            context.save_state(agent)
            context.messaging().reply(message, {"text": model_response})

    except KeyboardInterrupt:
        logger.info("Shutting down chatbot agent")
    except Exception as e:
        logger.error(f"Error in chatbot agent: {e}", exc_info=True)
        context.mark_failed(str(e))
    finally:
        context.save_state(agent)

if __name__ == "__main__":
    main()
