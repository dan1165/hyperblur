from typing import NamedTuple


class LoggingConfig(NamedTuple):
    """NamedTuple that stores configuration values relating to logging

    Attributes:
        sanic_logging_level:
            Numerical log level for the underlying server framework (Sanic)
        openblur_logging_level:
            Numerical log level for openblur
        openblur_extractor_logging_level:
            Numerical log level for openblur's extractor backend
    """

    sanic_logging_level: int = 50
    openblur_logging_level: int = 30
    openblur_extractor_logging_level: int = 30
