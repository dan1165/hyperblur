import sys
import tomllib
from typing import NamedTuple, Optional

from .dotenv import load_dotenv


class DeploymentConfig(NamedTuple):
    """Configuration values relating to deployment"""

    host: str = "127.0.0.1"
    port: int = 8080
    domain: Optional[str] = None

    # Enables secure cookies and forces all links to use the https:// scheme
    https: bool = False

    # Amount of worker instances to spawn. Left at 1, Sanic's fast mode spawns
    # one worker per CPU core automatically.
    workers: int = 1


class BackendConfig(NamedTuple):
    """Configuration values relating to how openblur requests Tumblr"""

    main_response_timeout: int = 10
    image_response_timeout: int = 30


class DefaultUserPreferences(NamedTuple):
    """Default user preferences"""

    expand_posts: bool = False


class LoggingConfig(NamedTuple):
    """Logging levels for Sanic, openblur, and the extractor"""

    sanic_logging_level: int = 50
    openblur_logging_level: int = 30
    openblur_extractor_logging_level: int = 30


class MiscellaneousConfig(NamedTuple):
    """Configuration values that don't fit anywhere else"""

    dev_mode: bool = False


class Config(NamedTuple):
    deployment: DeploymentConfig
    backend: BackendConfig
    default_user_preferences: DefaultUserPreferences
    logging: LoggingConfig
    misc: MiscellaneousConfig


# (config object, field name on Config, section name in config.toml)
_SECTIONS = (
    (DeploymentConfig, "deployment", "deployment"),
    (BackendConfig, "backend", "openblur_backend"),
    (DefaultUserPreferences, "default_user_preferences", "default_user_preferences"),
    (LoggingConfig, "logging", "logging"),
    (MiscellaneousConfig, "misc", "misc"),
)


def load_config(path: str) -> Config:
    """Loads a TOML configuration file into a Config object"""

    # Load a .env file (if present) so that optional secrets such as the
    # Tumblr API token can be supplied without editing the configuration file.
    load_dotenv()

    try:
        with open(path, "rb") as config_file:
            raw_config = tomllib.load(config_file)
    except FileNotFoundError:
        print(
            'Cannot find configuration file at "config.toml". '
            'Did you mean to set a new location with the environmental variable "OPENBLUR_CONFIG_LOCATION"?'
        )
        sys.exit()
    except PermissionError:
        print("Cannot access the configuration file. Do I have the right permissions?")
        sys.exit()

    # The config file can contain additional arguments that openblur does not
    # recognize, so only retrieve the fields each section understands.
    return Config(
        **{
            field: section_class(
                **{
                    key: value
                    for key, value in raw_config.get(section, {}).items()
                    if key in section_class._fields
                }
            )
            for section_class, field, section in _SECTIONS
        }
    )
