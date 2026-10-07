async def get_poll_results(ctx, blog, post_id, poll_id, expired=False):
    """Fetches poll results straight from Tumblr

    openblur does not cache anything, so results are always fresh.
    """
    return await _fetch_poll_results(ctx.TumblrAPI, blog, post_id, poll_id)


async def _fetch_poll_results(tumblr_api, blog, post_id, poll_id):
    """Requests Tumblr for poll results"""
    initial_results = await tumblr_api.poll_results(blog, post_id, poll_id)
    return initial_results["response"]
