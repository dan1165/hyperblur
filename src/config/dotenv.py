"""Minimal .env file loader

Populates os.environ with KEY=VALUE pairs read from a .env file. Variables
that are already present in the environment always take precedence, so real
environment variables (e.g. those set by Docker or the shell) are never
overwritten.
"""

import os

DEFAULT_ENV_PATH = ".env"


def _strip_quotes(value: str) -> str:
    if len(value) >= 2 and value[0] == value[-1] and value[0] in ("'", '"'):
        return value[1:-1]
    return value


def load_dotenv(path: str = DEFAULT_ENV_PATH) -> None:
    """Loads a .env file into os.environ if it exists"""
    try:
        with open(path) as env_file:
            lines = env_file.readlines()
    except OSError:
        # Missing/unreadable .env files are fine. They are entirely optional.
        return

    for line in lines:
        line = line.strip()

        if not line or line.startswith("#") or "=" not in line:
            continue

        if line.startswith("export "):
            line = line[len("export ") :]

        key, _, value = line.partition("=")
        key = key.strip()
        value = _strip_quotes(value.strip())

        if key and key not in os.environ:
            os.environ[key] = value
