import json
import os
import sys
from typing import Dict, Any, List, Optional

from agent_sdk.runtime.context import get_agent_context
from agent_sdk.runtime.entrypoint import get_runtime
from agent_sdk.runtime.model import get_model_provider

# Initialize the agent runtime
runtime = get_runtime()
agent_context = get_agent_context()

# Initialize the LLM with Gemini Flash 2.0
MODEL_NAME = os.environ.get("MODEL_NAME", "gemini-pro")
model_provider = get_model_provider("gemini")

# History container for conversation persistence
CONVERSATION_KEY = "conversation_history"

def initialize():
    """Initialize the agent state."""
    if not agent_context.get_value(CONVERSATION_KEY):
        agent_context.set_value(CONVERSATION_KEY, [])
    print(f"Chatbot agent initialized with model: {MODEL_NAME}")

def handle_message(message: Dict[str, Any]) -> Dict[str, Any]:
    """Handle an incoming message by sending it to the LLM and returning the response."""
    try:
        # Extract the message payload
        if isinstance(message, str):
            payload = json.loads(message)
        else:
            payload = message
            
        user_message = payload.get("text", "")
        if not user_message:
            return {"error": "No message text provided"}
        
        print(f"Received message: {user_message}")
        
        # Get conversation history
        conversation_history = agent_context.get_value(CONVERSATION_KEY) or []
        
        # Add user message to history
        conversation_history.append({"role": "user", "content": user_message})
        
        # Prepare context for the LLM
        messages = format_messages_for_model(conversation_history)
        
        # Call the LLM
        response = model_provider.generate_text(
            model=MODEL_NAME,
            messages=messages
        )
        
        # Extract the response text
        response_text = response.text if hasattr(response, 'text') else str(response)
        
        # Add assistant's response to history
        conversation_history.append({"role": "assistant", "content": response_text})
        
        # Update the conversation history in the agent context
        agent_context.set_value(CONVERSATION_KEY, conversation_history)
        
        # Construct the response
        response_payload = {
            "text": response_text,
            "conversation_length": len(conversation_history) // 2  # Number of turns
        }
        
        print(f"Sending response: {response_text[:100]}{'...' if len(response_text) > 100 else ''}")
        return response_payload
        
    except Exception as e:
        print(f"Error processing message: {str(e)}", file=sys.stderr)
        return {"error": str(e)}

def format_messages_for_model(conversation_history: List[Dict[str, str]]) -> List[Dict[str, str]]:
    """Format the conversation history for the specific model provider."""
    # For Gemini, we can use the history directly as it understands "user" and "assistant" roles
    return conversation_history

# Register the agent message handler
runtime.register_handler(initialize, handle_message)

# Start the agent
if __name__ == "__main__":
    runtime.start()
