from .base import AccessCache
from .. import openblur_extractor


class BlogPostsCache(AccessCache):
    def __init__(self, ctx, blog, continuation, **kwargs):
        super().__init__(ctx=ctx, continuation=continuation, **kwargs)
        self.blog = blog

    async def fetch(self):
        """Fetches blog posts from Tumblr"""
        return await self.ctx.TumblrAPI.blog_posts(
            self.blog, continuation=self.continuation, **self.kwargs
        )

    def parse(self, initial_results):
        return openblur_extractor.parse_blog_timeline(initial_results)


class BlogPostCache(AccessCache):
    def __init__(self, ctx, blog, post_id, **kwargs):
        super().__init__(ctx=ctx, **kwargs)

        self.blog = blog
        self.post_id = post_id

    async def fetch(self):
        return await self.ctx.TumblrAPI.blog_post(self.blog, self.post_id, **self.kwargs)

    def parse(self, initial_results):
        return openblur_extractor.parse_timeline(initial_results)


class BlogSearchCache(BlogPostsCache):
    def __init__(self, ctx, blog, query, continuation, **kwargs):
        super().__init__(ctx=ctx, blog=blog, continuation=continuation, **kwargs)

        self.query = query

    async def fetch(self):
        return await self.ctx.TumblrAPI.blog_search(
            self.blog, self.query, continuation=self.continuation, **self.kwargs
        )

    def parse(self, initial_results):
        return openblur_extractor.parse_blog_timeline(initial_results, is_search=True)


async def get_blog_posts(ctx, blog, continuation=None, **kwargs):
    blog_posts_cache = BlogPostsCache(ctx, blog, continuation, **kwargs)
    return await blog_posts_cache.get()


async def get_blog_search_results(ctx, blog, query, continuation=None, **kwargs):
    """Gets search results from a blog, freshly fetched from Tumblr"""
    blog_search_cache = BlogSearchCache(ctx, blog, query, continuation, **kwargs)
    return await blog_search_cache.get()


async def get_blog_post(ctx, blog, post_id, **kwargs):
    blog_post_cache = BlogPostCache(ctx, blog, post_id, **kwargs)
    return await blog_post_cache.get()
