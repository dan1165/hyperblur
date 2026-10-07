from openblur_extractor import openblur_exceptions


async def tumblr_error_login_walled(request, exception):
    return await request.app.ctx.render(
        "misc/msg_error",
        context={
            "app": request.app,
            "exception": exception,
            "error_heading": request.app.ctx.translate(
                "tumblr_error_blog_login_required_error_heading"
            ),
            "error_description": request.app.ctx.translate(
                "tumblr_error_blog_login_required_error_description"
            ),
        },
        status=403,
    )


async def tumblr_password_required_blog(request, exception):
    return await request.app.ctx.render(
        "misc/msg_error",
        context={
            "app": request.app,
            "exception": exception,
            "error_heading": request.app.ctx.translate(
                "tumblr_error_blog_requires_password_error_heading"
            ),
            "error_description": request.app.ctx.translate(
                "tumblr_error_blog_login_required_error_description"
            ),
        },
        status=403,
    )


async def tumblr_error_restricted_tag(request, exception):
    return await request.app.ctx.render(
        "misc/msg_error",
        context={
            "app": request.app,
            "exception": exception,
            "error_heading": request.app.ctx.translate("tumblr_error_restricted_tag_error_heading"),
            "error_description": request.app.ctx.translate(
                "tumblr_error_restricted_tag_description"
            ),
        },
        status=403,
    )


async def tumblr_error_unknown_blog(request, exception):
    return await request.app.ctx.render(
        "misc/msg_error",
        context={
            "app": request.app,
            "exception": exception,
            "error_heading": request.app.ctx.translate("tumblr_error_blog_not_found_error_heading"),
            "error_description": request.app.ctx.translate(
                "tumblr_error_blog_not_found_error_description"
            ),
        },
        status=404,
    )


async def tumblr_error_debug_non_json_response_error(request, exception):
    return await request.app.ctx.render(
        "misc/msg_error",
        context={
            "app": request.app,
            "exception": exception,
            "error_heading": f"Non 200 status code. Tumblr returned {exception.status_code} ",
            "error_description": "Tumblr returned an unexpected response. Please try again later.",
        },
        status=500,
    )


TUMBLR_ERROR_HANDLERS = {
    openblur_exceptions.TumblrLoginRequiredError: tumblr_error_login_walled,
    openblur_exceptions.TumblrPasswordRequiredBlogError: tumblr_password_required_blog,
    openblur_exceptions.TumblrRestrictedTagError: tumblr_error_restricted_tag,
    openblur_exceptions.TumblrBlogNotFoundError: tumblr_error_unknown_blog,
    openblur_exceptions.TumblrNon200NorJSONResponse: tumblr_error_debug_non_json_response_error,
}
