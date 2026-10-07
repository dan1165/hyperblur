import sanic.log

GENERIC_FORMAT = "%(asctime)s - (%(name)s) [%(process)d] [%(levelname)s]: %(message)s "


def setup_logging(logging_config):
    """Setup Sanic's logging configuration"""
    sanic_logging_config = sanic.log.LOGGING_CONFIG_DEFAULTS.copy()

    # Set Sanic's own logging to the desired logging level
    for logger in sanic_logging_config["loggers"].values():
        logger["level"] = logging_config.sanic_logging_level

    # A generic openblur console handler and formatter
    formatter = sanic_logging_config["formatters"]["generic"].copy()
    formatter["format"] = GENERIC_FORMAT
    sanic_logging_config["formatters"]["openblur_generic"] = formatter

    handler = sanic_logging_config["handlers"]["console"].copy()
    handler["formatter"] = "openblur_generic"
    sanic_logging_config["handlers"]["openblur_generic_console"] = handler

    sanic_logging_config["loggers"]["openblur"] = {
        "level": logging_config.openblur_logging_level,
        "handlers": ["openblur_generic_console"],
        "propagate": True,
        "qualname": "openblur",
    }

    sanic_logging_config["loggers"]["openblur-extractor"] = {
        "level": logging_config.openblur_extractor_logging_level,
        "handlers": ["openblur_generic_console"],
        "propagate": True,
        "qualname": "openblur-extractor",
    }

    return sanic_logging_config
