from .base import AccessCache
from .. import openblur_extractor


class ExploreCache(AccessCache):
    def __init__(self, ctx, type_, continuation, fetch_function, **kwargs):
        super().__init__(ctx=ctx, continuation=continuation, **kwargs)
        self.fetch_function = fetch_function

    async def fetch(self):
        """Fetches explore results from Tumblr"""
        return await self.fetch_function(continuation=self.continuation, **self.kwargs)

    def parse(self, initial_results):
        return openblur_extractor.parse_timeline(initial_results)


async def get_explore_results(ctx, fetch_function, type_, continuation, **kwargs):
    explore_cache = ExploreCache(ctx, type_, continuation, fetch_function, **kwargs)
    return await explore_cache.get()
