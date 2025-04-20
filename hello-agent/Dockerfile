FROM python:3.11-slim

WORKDIR /app

# Install system dependencies for psycopg2
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libpq-dev \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies (including google-adk)
RUN pip install --no-cache-dir psycopg2-binary redis google-adk

# Copy agent_sdk wheel and install as a package
COPY agent_sdk-*.whl /app/
RUN pip install /app/agent_sdk-*.whl

COPY main.py /app/main.py
RUN chmod +x /app/main.py

ENV PYTHONPATH=/app

CMD ["python", "-m", "agent_sdk.runtime.entrypoint"]
