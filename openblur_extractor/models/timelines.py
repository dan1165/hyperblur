from typing import Sequence, Optional, NamedTuple

from . import base
from .post import Post, ReplyNote, ReblogNote, LikeNote
from .misc import Signpost
from .blog import Blog


class BlogTimeline(NamedTuple):
    """Object representing a blog page

    TODO better documentation
    """

    blog_info: Blog
    posts: Sequence[Post]
    total_posts: int | None
    next: Optional[base.Cursor] = None


class NoteTimeline(NamedTuple):
    notes: Sequence[ReplyNote | ReblogNote | LikeNote]

    total_notes: int
    total_replies: int
    total_reblogs: int
    total_likes: int

    # Used to fetch next batch of post notes
    #
    # Reblogs and likes both use before_timestamp
    # but replies uses after_id
    before_timestamp: Optional[str] = None
    after_id: Optional[str] = None


class Timeline(NamedTuple):
    """Object representing Tumblr API's Timeline object.

    Refers to data on a certain page. IE Search or explore
    """

    elements: Sequence[Post | Blog]
    signposts: Sequence[Signpost] = []
    next: Optional[base.Cursor] = None
