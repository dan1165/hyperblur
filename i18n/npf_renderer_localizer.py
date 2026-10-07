import functools

import babel.dates
import babel.numbers


class NPFRendererLocalizer:
    """Bridges openblur's English strings and Babel formatting to npf-renderer's dict API"""

    def __init__(self, language, translate_func) -> None:
        self.language = language
        self.translate_func = translate_func

        self.formatting = {
            "duration": {
                "__default__": functools.partial(
                    babel.dates.format_timedelta, threshold=1.1, locale=language
                )
            },
            "datetime": {
                "__default__": functools.partial(
                    babel.dates.format_datetime, format="short", locale=language
                )
            },
            "decimal": {
                "__default__": functools.partial(babel.numbers.format_decimal, locale=language)
            },
        }

    def __getitem__(self, key: str):
        if key == "strings":
            return self
        if key == "formats":
            return self.formatting

        if key.startswith("plural_"):
            return lambda number: self.translate_func(f"npf_renderer_{key[7:]}", number)

        return self.translate_func(f"npf_renderer_{key}")
