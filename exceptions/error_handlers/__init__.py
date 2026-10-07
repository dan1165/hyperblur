from .base import create_user_friendly_error_message
from .extractor_errors import TUMBLR_ERROR_HANDLERS
from .miscellaneous_errors import MISCELLANEOUS_ERROR_HANDLERS


def register(app):
    """Registers all known error handlers into the given Sanic application"""
    for handlers in (TUMBLR_ERROR_HANDLERS, MISCELLANEOUS_ERROR_HANDLERS):
        for exception, handler in handlers.items():
            app.error_handler.add(exception, handler)
