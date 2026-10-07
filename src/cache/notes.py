from .base import AccessCache
from .. import openblur_extractor


class NotesTimelineCache(AccessCache):
    def __init__(self, ctx, blog, post_id, type_, fetch_function, **kwargs):
        super().__init__(
            ctx=ctx,
            continuation=kwargs.get("after_id") or kwargs.get("before_timestamp") or None,
            **kwargs,
        )

        self.blog = blog
        self.post_id = post_id

        self.type_ = type_
        self.fetch_function = fetch_function

    async def fetch(self):
        """Fetches notes from Tumblr"""
        return await self.fetch_function(self.blog, self.post_id, **self.kwargs)

    def parse(self, initial_results):
        return openblur_extractor.parse_note_timeline(initial_results)


async def get_post_notes(ctx, blog: str, post_id: str, type_: str, fetch_function, **kwargs):
    return await NotesTimelineCache(ctx, blog, post_id, type_, fetch_function, **kwargs).get()
