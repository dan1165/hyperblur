from typing import NamedTuple, Optional


class HeaderInfo(NamedTuple):
    header_image: str
    focused_header_image: str
    scaled_header_image: str


class BlogTheme(NamedTuple):
    avatar_shape: str
    background_color: Optional[str] = None
    body_font: Optional[str] = None
    header_info: Optional[HeaderInfo] = None


class BrokenBlog(NamedTuple):
    name: str
    avatar: list[dict]


class Blog(NamedTuple):
    name: str
    # [{"width": 512, "height": 512, url: "..."}, {"width": ...}...]
    avatar: list[dict]
    title: str
    url: str
    is_adult: bool

    description_npf: list[dict]
    uuid: str
    theme: BlogTheme

    # If blog is deactivated or not
    active: bool = False

    # Whether or not the blog requires an account to access
    requires_account_to_view: Optional[bool] = False
