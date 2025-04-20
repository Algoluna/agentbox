"""
ModelManager for agent SDK.
Wrapper around Google ADK's model functionality for text, multimodal, and structured outputs.
"""

import os
import logging
from typing import Optional, Any, Dict

from google.adk.generative_models import GenerativeModel, GenerationConfig, Content

class ModelManager:
    def __init__(self, model_name: Optional[str] = None):
        self.logger = logging.getLogger("ModelManager")
        self.model_name = model_name or os.environ.get("MODEL_NAME", "models/gemini-flash-2.0")
        self.model = GenerativeModel(self.model_name)
        self.logger.info(f"Initialized ModelManager with model: {self.model_name}")

    def ask(self, prompt: str, schema: Optional[Dict[str, Any]] = None) -> str:
        """
        Send a prompt to the model and get a response.

        Args:
            prompt: The prompt to send to the model
            schema: Optional response schema (for structured outputs)

        Returns:
            The model's response text
        """
        try:
            content = Content.from_text(prompt)
            generation_config = GenerationConfig(
                temperature=0.7,
                top_p=0.95,
                top_k=40,
                max_output_tokens=1024,
            )
            if schema:
                response = self.model.generate_content(
                    content,
                    generation_config=generation_config,
                    response_schema=schema
                )
            else:
                response = self.model.generate_content(
                    content,
                    generation_config=generation_config
                )
            return response.text
        except Exception as e:
            self.logger.error(f"Error calling model: {str(e)}")
            return f"Error: {str(e)}"
