"""Minimal .env loader

Populates os.environ from a .env file. Variables already present in the
environment (e.g. set by Docker or the shell) are never overwritten.
"""

import os


def load_dotenv(path: str = ".env") -> None:
    try:
        with open(path) as env_file:
            lines = env_file.readlines()
    except OSError:
        # Missing or unreadable .env files are fine; they are optional.
        return

    for line in lines:
        line = line.strip()

        if not line or line.startswith("#") or "=" not in line:
            continue

        key, _, value = line.partition("=")
        key = key.strip()

        if key and key not in os.environ:
            os.environ[key] = value.strip().strip("'\"")
