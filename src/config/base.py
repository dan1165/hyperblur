import sys
import tomllib

from typing import NamedTuple

from . import deployment, openblur_backend, user_preferences, logging_config, misc
from .dotenv import load_dotenv


class openblurConfig(NamedTuple):
    """NamedTuple storing configuration data for openblur

    Encapsulates various configuration settings under a single field.

    Attributes:
        deployment: Configuration settings for deploying openblur
        backend: Configuration settings to customize
            how openblur requests Tumblr
        logging: Configuration settings to change logging behavior
        misc: Configuration settings that doesn't fit into any other categories
    """

    deployment: deployment.DeploymentConfig
    backend: openblur_backend.openblurBackendConfig
    default_user_preferences: user_preferences.DefaultUserPreferences
    logging: logging_config.LoggingConfig
    misc: misc.MiscellaneousConfig


def load_config(path: str) -> openblurConfig:
    """Loads a TOML configuration file into a openblurConfig object"""

    # Load a .env file (if present) so that optional secrets such as the
    # Tumblr API token can be supplied without editing the configuration file.
    load_dotenv()

    try:
        with open(path, "rb") as config_file:
            config = tomllib.load(config_file)
    except FileNotFoundError:
        print(
            'Cannot find configuration file at "./config.toml". '
            'Did you mean to set a new location with the environmental variable "OPENBLUR_CONFIG_LOCATION"?'
        )
        sys.exit()
    except PermissionError:
        print("Cannot access the configuration file. Do I have the right permissions?")
        sys.exit()

    # The config file can contain additional arguments that openblur does not recognize.
    # As such some processing is needed to only retrieve what openblur can understand

    # Defines config sections
    config_sections = (
        # Corresponding object, internal name, section name in the config file
        (deployment.DeploymentConfig, "deployment", "deployment"),
        (openblur_backend.openblurBackendConfig, "backend", "openblur_backend"),
        (
            user_preferences.DefaultUserPreferences,
            "default_user_preferences",
            "default_user_preferences",
        ),
        (logging_config.LoggingConfig, "logging", "logging"),
        (misc.MiscellaneousConfig, "misc", "misc"),
    )

    openblur_config_data = {}

    for section_definition in config_sections:
        section_object, internal_name, external_name = section_definition
        arguments_to_load = {}
        arguments_from_config = config.get(external_name, {})

        # Ignore unknown config fields
        for k, v in arguments_from_config.items():
            if k in section_object._fields:
                arguments_to_load[k] = v

        openblur_config_data[internal_name] = section_object(**arguments_to_load)

    # TODO Validate invalid config values

    return openblurConfig(**openblur_config_data)
