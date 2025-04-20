# agent_sdk.runtime.embedding.py

"""
EmbeddingManager stub for agent SDK.
Backend-agnostic interface for embedding/vector store access (future extension).
"""

class EmbeddingManager:
    def __init__(self):
        # TODO: Initialize embedding backend if needed
        pass

    def embed(self, text: str):
        # TODO: Call embedding backend with text
        raise NotImplementedError
