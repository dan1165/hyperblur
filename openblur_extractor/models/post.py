import datetime
import enum

from typing import Optional, Union, NamedTuple, Sequence

from . import blog


class CommunityLabel(enum.Enum):
    MATURE = 0  # Generic catch all
    DRUG_USE = 1
    VIOLENCE = 2
    SEXUAL_THEMES = 3


class ReplyNote(NamedTuple):
    uuid: str
    reply_id: str
    date: Optional[datetime.datetime]

    content: Optional[Sequence[dict]]
    layout: Optional[Sequence[dict]]

    blog: blog.Blog


class ReblogNote(NamedTuple):
    uuid: str
    id: str

    blog: blog.Blog

    content: Optional[Sequence[dict]]
    layout: Optional[Sequence[dict]]
    tags: Sequence[str]

    reblogged_from: str

    date: Optional[datetime.datetime]

    community_labels: Sequence[CommunityLabel]


class LikeNote(NamedTuple):
    blog_name: str
    blog_uuid: str
    blog_title: str
    date: Optional[datetime.datetime]

    avatar: list[dict]

    # TODO
    # avatar_shape


class ReblogAttribution(NamedTuple):
    """Object representing reblog author information from individual posts"""

    post_id: str
    post_url: str
    blog_name: str
    blog_title: str


class PostTrail(NamedTuple):
    id: Optional[str]
    blog: Union[blog.Blog, blog.BrokenBlog]
    date: Optional[datetime.datetime]
    content: Optional[list[dict]]
    layout: Optional[list[dict]]


class Post(NamedTuple):
    blog: blog.Blog

    id: str
    post_url: str
    slug: str
    date: Optional[datetime.datetime]
    tags: list[str]
    summary: str

    display_avatar: bool
    # intractability: str TODO

    is_advertisement: bool
    is_nsfw: bool

    content: Optional[list[dict]]
    layout: Optional[list[dict]]
    trail: Sequence[PostTrail]

    note_count: Optional[int] = None
    like_count: Optional[int] = None
    reblog_count: Optional[int] = None
    reply_count: Optional[int] = None

    default_note_viewer_tab: str = "replies"

    reblog_from: Optional[ReblogAttribution] = None
    reblog_root: Optional[ReblogAttribution] = None

    community_labels: list[CommunityLabel] = []
