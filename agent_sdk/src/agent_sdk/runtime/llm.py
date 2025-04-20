# agent_sdk.runtime.llm.py

"""
LLMManager stub for agent SDK.
Backend-agnostic interface for LLM access (future extension).
"""

class LLMManager:
    def __init__(self):
        # TODO: Initialize LLM backend if needed
        pass

    def ask(self, prompt: str, schema=None):
        # TODO: Call LLM backend with prompt (optionally with schema)
        raise NotImplementedError
