from .base import AccessCache
from .. import openblur_extractor


class SearchCache(AccessCache):
    def __init__(self, ctx, query, continuation, **kwargs):
        super().__init__(ctx=ctx, continuation=continuation, **kwargs)

        self.query = query

    async def fetch(self):
        """Fetches search results from Tumblr"""
        return await self.ctx.TumblrAPI.timeline_search(
            self.query,
            self.ctx.TumblrAPI.config.TimelineType.POST,
            continuation=self.continuation,
            **self.kwargs,
        )

    def parse(self, initial_results):
        return openblur_extractor.parse_timeline(initial_results)


async def get_search_results(ctx, query, continuation=None, **kwargs):
    search_cache = SearchCache(ctx, query, continuation, **kwargs)
    return await search_cache.get()
