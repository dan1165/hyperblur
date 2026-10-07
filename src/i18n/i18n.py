from .npf_renderer_localizer import NPFRendererLocalizer
from .strings import PLURALS, STRINGS


def translate(id, number=None, substitution=None):
    if number is not None and id in PLURALS:
        singular, plural = PLURALS[id]
        text = singular if number == 1 else plural
    else:
        text = STRINGS.get(id, id)

    if isinstance(substitution, str):
        text = text.format(substitution)
    elif isinstance(substitution, dict):
        text = text.format(**substitution)

    return text


class Language:
    """Holds the localized helpers for a locale"""

    def __init__(self, locale) -> None:
        self.locale = locale
        self.npf_renderer_localizer = NPFRendererLocalizer(locale, translate)


def initialize_locales() -> dict:
    """openblur only ships an English locale"""
    return {"en_US": Language("en_US")}
