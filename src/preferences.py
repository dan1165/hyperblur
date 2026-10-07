import dataclasses
import urllib.parse


@dataclasses.dataclass
class UserPreferences:
    expand_posts: bool = True

    def replace_from_forms(self, request) -> "UserPreferences":
        """Returns updated UserPreferences from POST form data"""
        return self._replace(request.form)

    def replace_from_query(self, request) -> "UserPreferences":
        """Returns updated UserPreferences from request query args"""
        return self._replace(request.args)

    def replace_from_cookie(self, request) -> "UserPreferences":
        """Returns updated UserPreferences from the settings cookie"""
        request.ctx.invalid_settings_cookie = False

        try:
            if raw_prefs := request.cookies.get("settings"):
                return self._replace(urllib.parse.parse_qs(raw_prefs))
        except (TypeError, KeyError, ValueError):
            request.ctx.invalid_settings_cookie = True

        return self

    def _replace(self, raw_prefs):
        value = raw_prefs.get("expand_posts")
        if not value:
            return self

        if isinstance(value, str):
            value = [value]

        return dataclasses.replace(self, expand_posts=value[0] == "on")

    def to_url_encoded(self):
        """Encodes the preferences as URL query parameters

        Used to restore settings at /settings/restore
        """
        return urllib.parse.urlencode({"expand_posts": "on" if self.expand_posts else "off"})

    def construct_cookie(self):
        """Serializes user preferences into a cookie"""
        return {
            "key": "settings",
            "value": self.to_url_encoded(),
            "max_age": 31540000,
        }
