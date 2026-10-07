from typing import NamedTuple


class DefaultUserPreferences(NamedTuple):
    """NamedTuple that stores default user Preferences

    Attributes:
        expand_posts: whether to expand truncated posts by default
    """

    expand_posts: bool = False
