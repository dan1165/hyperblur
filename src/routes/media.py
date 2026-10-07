import asyncio
import re
import urllib.parse

import aiohttp
import sanic

media = sanic.Blueprint("TumblrMedia", url_prefix="/tblr")

# A CDN label is the first label of a `*.media.tumblr.com` host. Restricting
# it keeps crafted requests from ever pointing the redirect/proxy elsewhere.
_CDN_NAME_PATTERN = re.compile(r"^[a-zA-Z0-9-]{1,63}$")

# Media on Tumblr's CDN is immutable, so redirects to it can be cached for a
# long time. This lets repeat visits skip Priviblur entirely.
_MEDIA_CACHE_CONTROL = "public, max-age=2629800, immutable"

# Headers that must never be forwarded verbatim from the client to Tumblr.
_FORWARDED_REQUEST_HEADERS = ("range", "if-range", "if-none-match", "if-modified-since")

_MAX_FILENAME_LENGTH = 150


def _is_download_requested(request: sanic.Request) -> bool:
    try:
        return sanic.utils.str_to_bool(request.args.get("download", ""))
    except ValueError:
        return False


def _quote_path(path: str) -> str:
    return urllib.parse.quote(path, safe="/")


def _validate_path(path: str):
    """Rejects path traversal attempts while still allowing nested paths"""
    decoded_path = urllib.parse.unquote(path)

    if "\x00" in decoded_path or ".." in decoded_path.split("/"):
        raise sanic.exceptions.NotFound("Invalid media path")


def _redirect_to_origin(origin_url: str) -> sanic.HTTPResponse:
    response = sanic.redirect(origin_url, status=302)
    response.headers["Cache-Control"] = _MEDIA_CACHE_CONTROL
    return response


def _build_content_disposition(path: str) -> str:
    """Builds a safe Content-Disposition header for forced downloads"""
    filename = urllib.parse.unquote(path).rsplit("/", 1)[-1]
    filename = "".join(char for char in filename if char not in '"\\\r\n')[:_MAX_FILENAME_LENGTH]

    if not filename:
        filename = "download"

    ascii_filename = filename.encode("ascii", "ignore").decode() or "download"
    encoded_filename = urllib.parse.quote(filename)

    return f"attachment; filename=\"{ascii_filename}\"; filename*=UTF-8''{encoded_filename}"


async def stream_media(
    request: sanic.Request,
    client: aiohttp.ClientSession,
    path_to_request: str,
    additional_headers: dict | None = None,
    base_url: str = "",
    download_filename: str | None = None,
):
    """Streams media from Tumblr through Priviblur

    This is only used when a forced download is requested. Normal media loads
    are redirected straight to Tumblr's CDN (see `_media_cdn` and friends).
    """
    request_headers = dict(additional_headers or {})

    # Forward conditional/range headers so partial downloads and resumable
    # transfers work.
    for header in _FORWARDED_REQUEST_HEADERS:
        if value := request.headers.get(header):
            request_headers[header] = value

    try:
        async with client.get(f"{base_url}/{path_to_request}", headers=request_headers) as upstream:
            response_headers = {
                key: value
                for key, value in upstream.headers.items()
                if key.lower() not in request.app.ctx.BLACKLIST_RESPONSE_HEADERS
            }

            if download_filename:
                response_headers["Content-Disposition"] = download_filename

            response = await request.respond(status=upstream.status, headers=response_headers)

            try:
                async for chunk in upstream.content.iter_any():
                    await response.send(chunk)
            except (aiohttp.ClientError, asyncio.TimeoutError, OSError) as exception:
                request.app.ctx.LOGGER.warning(
                    "Media stream interrupted for %s: %s", request.path, exception
                )
            finally:
                await response.eof()

            return response
    except (aiohttp.ClientError, asyncio.TimeoutError, OSError) as exception:
        request.app.ctx.LOGGER.warning(
            "Failed to fetch media from Tumblr for %s: %s", request.path, exception
        )
        return sanic.response.empty(status=502)


@media.get("/media/<cdn:str>/<path:path>")
async def _media_cdn(request: sanic.Request, cdn: str, path: str):
    """Redirects requests for *.media.tumblr.com media to Tumblr's CDN

    When `?download=1` is set the media is proxied instead so that it can be
    served as an attachment.
    """
    if not _CDN_NAME_PATTERN.match(cdn):
        raise sanic.exceptions.NotFound("Unknown media CDN")

    _validate_path(path)

    if not _is_download_requested(request):
        return _redirect_to_origin(f"https://{cdn}.media.tumblr.com/{_quote_path(path)}")

    content_disposition = _build_content_disposition(path)

    video_headers = {
        "accept": "video/webm,video/ogg,video/*;q=0.9, application/ogg;q=0.7,audio/*;q=0.6,*/*;q=0.5"
    }

    match cdn:
        case "64":
            client = request.app.ctx.Media64Client
        case "49":
            client = request.app.ctx.Media49Client
        case "44":
            client = request.app.ctx.Media44Client
        case "ve":
            return await stream_media(
                request,
                request.app.ctx.MediaVeClient,
                path,
                additional_headers=video_headers,
                download_filename=content_disposition,
            )
        case "va":
            return await stream_media(
                request,
                request.app.ctx.MediaVaClient,
                path,
                additional_headers=video_headers,
                download_filename=content_disposition,
            )
        case _:
            return await stream_media(
                request,
                request.app.ctx.MediaGenericClient,
                path,
                base_url=f"https://{cdn}.media.tumblr.com",
                download_filename=content_disposition,
            )

    return await stream_media(request, client, path, download_filename=content_disposition)


@media.get(r"/a/<path:path>")
async def _a_media(request: sanic.Request, path: str):
    """Redirects requests for a.tumblr.com media to Tumblr's CDN"""
    _validate_path(path)

    if not _is_download_requested(request):
        return _redirect_to_origin(f"https://a.tumblr.com/{_quote_path(path)}")

    additional_headers = {
        "accept": "audio/webm,audio/ogg,audio/wav,audio/*;q=0.9,application/ogg;q=0.7,video/*;q=0.6,*/*;q=0.5"
    }
    return await stream_media(
        request,
        request.app.ctx.AudioClient,
        path,
        additional_headers=additional_headers,
        download_filename=_build_content_disposition(path),
    )


@media.get(r"/assets/<path:path>")
async def _tb_assets(request: sanic.Request, path: str):
    """Redirects requests for assets.tumblr.com to Tumblr's CDN"""
    _validate_path(path)

    if not _is_download_requested(request):
        return _redirect_to_origin(f"https://assets.tumblr.com/{_quote_path(path)}")

    return await stream_media(
        request,
        request.app.ctx.TumblrAssetClient,
        path,
        download_filename=_build_content_disposition(path),
    )


@media.get(r"/static/<path:path>")
async def _tb_static(request: sanic.Request, path: str):
    """Redirects requests for static.tumblr.com to Tumblr's CDN"""
    _validate_path(path)

    if not _is_download_requested(request):
        return _redirect_to_origin(f"https://static.tumblr.com/{_quote_path(path)}")

    return await stream_media(
        request,
        request.app.ctx.TumblrStaticClient,
        path,
        download_filename=_build_content_disposition(path),
    )
