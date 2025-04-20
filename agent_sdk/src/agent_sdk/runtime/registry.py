# agent_sdk.runtime.registry.py

"""
Agent registry and decorator for agent SDK.
Supports decorator-based agent registration and dynamic lookup.
"""

import logging

AGENT_REGISTRY = {}

def register_agent(name: str):
    def wrapper(agent_instance):
        AGENT_REGISTRY[name] = agent_instance
        logging.getLogger("agent-sdk-registry").info(f"Registered agent: {name} -> {agent_instance}")
        return agent_instance
    return wrapper

def get_registered_agent(name: str):
    return AGENT_REGISTRY.get(name)
