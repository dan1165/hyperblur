import logging

import sanic.log

GENERIC_FORMAT = "%(asctime)s - (%(name)s) [%(process)d] [%(levelname)s]: %(message)s "


def setup_logging():
    """Setup Sanic's logging configuration"""
    sanic_logging_config = sanic.log.LOGGING_CONFIG_DEFAULTS.copy()

    # Quiet Sanic's own loggers
    for logger in sanic_logging_config["loggers"].values():
        logger["level"] = logging.CRITICAL

    # A generic openblur console handler and formatter
    formatter = sanic_logging_config["formatters"]["generic"].copy()
    formatter["format"] = GENERIC_FORMAT
    sanic_logging_config["formatters"]["openblur_generic"] = formatter

    handler = sanic_logging_config["handlers"]["console"].copy()
    handler["formatter"] = "openblur_generic"
    sanic_logging_config["handlers"]["openblur_generic_console"] = handler

    sanic_logging_config["loggers"]["openblur"] = {
        "level": logging.WARNING,
        "handlers": ["openblur_generic_console"],
        "propagate": True,
        "qualname": "openblur",
    }

    sanic_logging_config["loggers"]["openblur-extractor"] = {
        "level": logging.WARNING,
        "handlers": ["openblur_generic_console"],
        "propagate": True,
        "qualname": "openblur-extractor",
    }

    return sanic_logging_config
