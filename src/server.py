import os
import logging
import urllib.parse
import functools

import sanic
import aiohttp
import orjson
import babel.numbers
import babel.dates
import babel.lists

from . import routes, openblur_extractor, preferences, i18n
from .exceptions import error_handlers
from .helpers import setup_logging, helpers, render, ext_npf_renderer


# openblur is configless: everything is fixed here, and the only thing read
# from the environment is the optional Tumblr API token.
HOST = "0.0.0.0"
PORT = 8000
DOMAIN = None
MAIN_REQUEST_TIMEOUT = 10
IMAGE_REQUEST_TIMEOUT = 30

app = sanic.Sanic(
    "openblur",
    loads=orjson.loads,
    dumps=orjson.dumps,
    log_config=setup_logging.setup_logging(),
)
app.config.OAS = False

# Performance: uvloop + httptools when installed, no access log or MOTD, and
# keep-alive so clients reuse the TLS connection.
app.config.USE_UVLOOP = True
app.config.ACCESS_LOG = False
app.config.MOTD = False
app.config.KEEP_ALIVE_TIMEOUT = 30

app.ctx.LANGUAGES = i18n.initialize_locales()

# Constants

app.config.TEMPLATING_PATH_TO_TEMPLATES = "src/templates"

app.ctx.LOGGER = logging.getLogger("openblur")

app.ctx.URL_HANDLER = helpers.url_handler
app.ctx.BLACKLIST_RESPONSE_HEADERS = ("access-control-allow-origin", "alt-svc", "server")

app.ctx.DOMAIN = DOMAIN
app.ctx.translate = i18n.translate

app.ctx.OPENBLUR_PARENT_DIR_PATH = os.path.abspath(os.path.dirname(os.path.dirname(__file__)))
app.ctx.create_user_friendly_error_message = error_handlers.create_user_friendly_error_message


@app.listener("before_server_start")
async def initialize(app):
    app.ctx.TumblrAPI = await openblur_extractor.TumblrAPI.create(
        main_request_timeout=MAIN_REQUEST_TIMEOUT, json_loads=orjson.loads
    )

    media_request_headers = {
        "user-agent": openblur_extractor.TumblrAPI.DEFAULT_HEADERS["user-agent"],
        "accept-encoding": "gzip, deflate",
        "accept": "image/avif,image/webp,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5",
        "accept-language": "en-US,en;q=0.5",
        "connection": "keep-alive",
        "te": "trailers",
        "referer": "https://www.tumblr.com/",
    }

    # TODO set pool size for image requests

    def create_image_client(url, timeout=IMAGE_REQUEST_TIMEOUT):
        timeout = aiohttp.ClientTimeout(timeout)
        return aiohttp.ClientSession(
            url,
            headers=media_request_headers,
            timeout=timeout,
            connector=openblur_extractor.helpers.create_connector(),
        )

    app.ctx.Media64Client = create_image_client("https://64.media.tumblr.com")

    app.ctx.Media49Client = create_image_client("https://49.media.tumblr.com")

    app.ctx.Media44Client = create_image_client("https://44.media.tumblr.com")

    app.ctx.MediaVeClient = create_image_client("https://ve.media.tumblr.com")

    app.ctx.MediaVaClient = create_image_client("https://va.media.tumblr.com")

    app.ctx.MediaGenericClient = aiohttp.ClientSession(
        headers=media_request_headers,
        timeout=aiohttp.ClientTimeout(IMAGE_REQUEST_TIMEOUT),
        connector=openblur_extractor.helpers.create_connector(),
    )

    app.ctx.AudioClient = create_image_client("https://a.tumblr.com")

    app.ctx.TumblrAssetClient = create_image_client("https://assets.tumblr.com")

    app.ctx.TumblrStaticClient = create_image_client("https://static.tumblr.com")

    app.ctx.TumblrAtClient = aiohttp.ClientSession(
        "https://at.tumblr.com",
        headers={"user-agent": openblur_extractor.TumblrAPI.DEFAULT_HEADERS["user-agent"]},
        timeout=aiohttp.ClientTimeout(MAIN_REQUEST_TIMEOUT),
        connector=openblur_extractor.helpers.create_connector(),
    )

    app.ctx.render = render.render_template

    # Add additional jinja filters and functions

    app.ext.environment.add_extension("jinja2.ext.do")

    app.ext.environment.filters["encodepathsegment"] = functools.partial(
        urllib.parse.quote, safe=""
    )

    app.ext.environment.filters["update_query_params"] = helpers.update_query_params
    app.ext.environment.filters["remove_query_params"] = helpers.remove_query_params
    app.ext.environment.filters["deseq_urlencode"] = functools.partial(
        urllib.parse.urlencode, doseq=True
    )
    app.ext.environment.filters["ensure_single_prefix_slash"] = (
        helpers.prefix_slash_in_url_if_missing
    )

    app.ext.environment.filters["format_decimal"] = babel.numbers.format_decimal
    app.ext.environment.filters["format_date"] = babel.dates.format_date
    app.ext.environment.filters["format_datetime"] = babel.dates.format_datetime

    app.ext.environment.filters["format_list"] = babel.lists.format_list

    app.ext.environment.globals["translate"] = i18n.translate
    app.ext.environment.globals["url_handler"] = helpers.url_handler
    app.ext.environment.globals["format_npf"] = ext_npf_renderer.format_npf
    app.ext.environment.globals["create_poll_callback"] = helpers.create_poll_callback
    app.ext.environment.globals["create_reblog_attribution"] = (
        helpers.create_reblog_attribution_link
    )

    app.ext.environment.tests["a_post"] = lambda element: isinstance(
        element, openblur_extractor.models.post.Post
    )


@app.listener("main_process_start")
async def main_startup_listener(app):
    """Startup listener to notify of openblur startup"""
    print("Starting up openblur")


@app.get("/")
async def root(request):
    return sanic.redirect(request.app.url_for("explore._trending"))


@app.route("/robots.txt")
async def robotstxt_route(request):
    return await sanic.file("./assets/robots.txt")


@app.middleware("request", priority=1)
async def before_all_routes(request):
    # openblur is English-only and always uses its dark theme.
    request.ctx.language = "en_US"

    request.ctx.preferences = preferences.UserPreferences()

    request.ctx.preferences = request.ctx.preferences.replace_from_cookie(request)


@app.middleware("response")
async def after_all_routes(request, response):
    # https://github.com/iv-org/invidious/blob/master/src/invidious/routes/before_all.cr
    response.headers["x-xss-protection"] = "1; mode=block"
    response.headers["x-content-type-options"] = "nosniff"
    response.headers["referrer-policy"] = "same-origin"

    # Never cache anything: every request is served fresh from the origin.
    response.headers["cache-control"] = "no-store, no-cache, must-revalidate, max-age=0"
    response.headers["pragma"] = "no-cache"
    response.headers["expires"] = "0"

    # Media is loaded directly from Tumblr's CDN (see src/routes/media.py), so
    # images and audio/video must be allowed to load from *.tumblr.com.
    response.headers["content-security-policy"] = "; ".join(
        [
            "default-src 'none'",
            "script-src 'self'",
            "style-src 'self' 'unsafe-inline'",
            "img-src 'self' data: https://*.tumblr.com",
            "font-src 'self' data:",
            "connect-src 'self'",
            "manifest-src 'self'",
            "media-src 'self' https://*.tumblr.com",
            "child-src 'self' blob:",
        ]
    )


# Register all routes:
for route in routes.BLUEPRINTS:
    app.blueprint(route)

# Register error handlers into openblur
error_handlers.register(app)

if __name__ == "__main__":
    # Sanic's fast mode spawns one worker per available CPU core and disables
    # access logs.
    app.run(host=HOST, port=PORT, access_log=False, fast=True)
