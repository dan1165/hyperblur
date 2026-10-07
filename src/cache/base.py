import abc


class AccessCache(abc.ABC):
    """Fetches and parses data straight from Tumblr

    openblur performs no caching. Every request hits the origin server so that
    the data returned is always fresh.
    """

    def __init__(self, ctx, continuation=None, **kwargs):
        self.ctx = ctx
        self.continuation = continuation
        self.kwargs = kwargs

    @abc.abstractmethod
    def fetch(self) -> dict:
        """Fetches results from Tumblr"""
        pass

    @abc.abstractmethod
    def parse(self, initial_results):
        """Parses the initial JSON response from Tumblr"""
        pass

    async def get(self):
        """Fetches fresh data from Tumblr and parses it"""
        initial_results = await self.fetch()
        return self.parse(initial_results)
