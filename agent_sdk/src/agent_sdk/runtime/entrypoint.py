# agent_sdk.runtime.entrypoint.py

"""
Standard entrypoint for ADK-compatible agents using the agent SDK.
- Reads secrets for Postgres and Valkey from files (with env fallback)
- Logs all secret values and connection attempts
- Sets up environment variables for the SDK
- Instantiates the registered agent and injects RuntimeContext
- Runs the main message loop
"""

import os
import logging
from agent_sdk.runtime.registry import get_registered_agent
from agent_sdk.runtime.context import RuntimeContext
from agent_sdk.runtime.messaging import Messaging
from agent_sdk.db.state import StateManager

def read_secret(path, env_var, default=None, required=False):
    try:
        with open(path, "r") as f:
            value = f.read().strip()
            if value:
                return value
    except Exception:
        pass
    value = os.environ.get(env_var, default)
    if required and not value:
        raise RuntimeError(f"Missing required secret or env: {path} or {env_var}")
    return value

def setup_logging():
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

def main():
    setup_logging()
    logger = logging.getLogger("agent-sdk-entrypoint")

    # Import agent modules as specified by AGENT_MODULES env var
    import importlib
    modules = os.environ.get("AGENT_MODULES", "main").split(",")
    for mod in modules:
        mod = mod.strip()
        if mod:
            logger.info(f"Importing agent module: {mod}")
            importlib.import_module(mod)

    # Read agent type and id
    agent_type = os.environ.get("AGENT_TYPE", "hello_agent")
    agent_id = os.environ.get("AGENT_ID", agent_type)

    # --- Postgres secrets ---
    pg_path = "/etc/secrets/postgres"
    db_username = read_secret(f"{pg_path}/username", "POSTGRES_USERNAME", required=True)
    db_password = read_secret(f"{pg_path}/password", "POSTGRES_PASSWORD", required=True)
    db_name = read_secret(f"{pg_path}/database", "POSTGRES_DB", required=True)
    db_host = read_secret(f"{pg_path}/host", "POSTGRES_HOST", required=True)
    db_port = read_secret(f"{pg_path}/port", "POSTGRES_PORT", default="5432", required=True)

    logger.info(f"Postgres secret values: username={db_username}, password={db_password}, database={db_name}, host={db_host}, port={db_port}")

    db_url = f"postgresql://{db_username}:{db_password}@{db_host}:{db_port}/{db_name}"
    os.environ["DB_URL"] = db_url

    # Test Postgres connection using StateManager
    try:
        state_mgr = StateManager(db_url)
        logger.info("Testing Postgres connection via StateManager...")
        state_mgr.save_state("__entrypoint_test__", {"test": True})
        logger.info("Successfully connected to Postgres and wrote test state.")
    except Exception as e:
        logger.error(f"Error connecting to Postgres via StateManager: {e}")

    # --- Valkey/Redis secrets ---
    valkey_path = "/etc/secrets/valkey"
    valkey_host = read_secret(f"{valkey_path}/host", "VALKEY_HOST", default="valkey", required=True)
    valkey_port = read_secret(f"{valkey_path}/port", "VALKEY_PORT", default="6379", required=True)
    valkey_password = read_secret(f"{valkey_path}/password", "VALKEY_PASSWORD", default=None, required=False)
    valkey_username = read_secret(f"{valkey_path}/username", "VALKEY_USERNAME", default=None, required=False)

    logger.info(f"Valkey secret values: host={valkey_host}, port={valkey_port}, username={valkey_username}, password={valkey_password}")

    if valkey_password:
        redis_url = f"redis://:{valkey_password}@{valkey_host}:{valkey_port}/0"
    else:
        redis_url = f"redis://{valkey_host}:{valkey_port}/0"
    os.environ["REDIS_URL"] = redis_url
    if valkey_username:
        os.environ["REDIS_USERNAME"] = valkey_username

    # Test Valkey/Redis connection using Messaging
    try:
        messaging = Messaging(redis_url, agent_type, agent_id)
        logger.info("Testing Valkey/Redis connection via Messaging...")
        # Try a ping by sending a dummy xadd to a test stream
        test_stream = f"agent:{agent_type}:{agent_id}:entrypoint_test"
        messaging.redis.xadd(test_stream, {"data": '{"test": true}'})
        logger.info("Successfully connected to Valkey/Redis and wrote test message.")
    except Exception as e:
        logger.error(f"Error connecting to Valkey/Redis via Messaging: {e}")

    # --- Agent instantiation ---
    agent_cls = get_registered_agent(agent_type)
    if agent_cls is None:
        logger.error(f"No agent registered for type: {agent_type}")
        exit(1)
    agent = agent_cls(name=agent_id)
    agent.ctx = RuntimeContext.from_env(agent.name)
    agent.ctx.load_state(agent)

    logger.info(f"Starting agent main loop (ID: {agent_id}, Type: {agent_type})")
    while True:
        msg = agent.ctx.receive()
        agent.on_message(msg)
        agent.ctx.save_state(agent)

if __name__ == "__main__":
    main()
