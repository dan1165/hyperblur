from .base import AccessCache
from .. import openblur_extractor


class TagBrowseCache(AccessCache):
    def __init__(self, ctx, tag, latest, continuation, **kwargs):
        super().__init__(ctx=ctx, continuation=continuation, **kwargs)

        self.tag = tag
        self.latest = latest

    async def fetch(self):
        """Fetches posts from Tumblr with the given tag"""
        return await self.ctx.TumblrAPI.hubs_timeline(
            self.tag, latest=self.latest, continuation=self.continuation
        )

    def parse(self, initial_results):
        return openblur_extractor.parse_timeline(initial_results)


async def get_tag_browse_results(ctx, tag, latest=False, continuation=None):
    tag_browse_cache = TagBrowseCache(ctx, tag, latest, continuation)
    return await tag_browse_cache.get()
