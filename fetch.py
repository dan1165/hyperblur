"""Fetches and parses data straight from Tumblr

openblur performs no caching: every request hits the origin server so the data
returned is always fresh.
"""

from openblur_extractor import (
    parse_timeline,
    parse_blog_timeline,
    parse_note_timeline,
)


async def get_poll_results(ctx, blog, post_id, poll_id, expired=False):
    """Fetches poll results straight from Tumblr"""
    results = await ctx.TumblrAPI.poll_results(blog, post_id, poll_id)
    return results["response"]


async def get_explore_results(ctx, fetch_function, type_, continuation, **kwargs):
    return parse_timeline(await fetch_function(continuation=continuation, **kwargs))


async def get_tag_browse_results(ctx, tag, latest=False, continuation=None):
    results = await ctx.TumblrAPI.hubs_timeline(tag, latest=latest, continuation=continuation)
    return parse_timeline(results)


async def get_search_results(ctx, query, continuation=None, **kwargs):
    results = await ctx.TumblrAPI.timeline_search(
        query, ctx.TumblrAPI.config.TimelineType.POST, continuation=continuation, **kwargs
    )
    return parse_timeline(results)


async def get_blog_posts(ctx, blog, continuation=None, **kwargs):
    results = await ctx.TumblrAPI.blog_posts(blog, continuation=continuation, **kwargs)
    return parse_blog_timeline(results)


async def get_blog_search_results(ctx, blog, query, continuation=None, **kwargs):
    results = await ctx.TumblrAPI.blog_search(blog, query, continuation=continuation, **kwargs)
    return parse_blog_timeline(results, is_search=True)


async def get_blog_post(ctx, blog, post_id, **kwargs):
    results = await ctx.TumblrAPI.blog_post(blog, post_id, **kwargs)
    return parse_timeline(results)


async def get_post_notes(ctx, blog, post_id, type_, fetch_function, **kwargs):
    results = await fetch_function(blog, post_id, **kwargs)
    return parse_note_timeline(results)
