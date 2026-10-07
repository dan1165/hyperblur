"""Shared SSL/TLS helpers for outbound requests to Tumblr"""

import ssl

import aiohttp


def create_ssl_context() -> ssl.SSLContext:
    """Creates an SSL context used for requests to Tumblr and its CDNs

    ALPN is intentionally left unset. aiohttp only speaks HTTP/1.1 and thus
    only ever advertises "http/1.1" through ALPN. Some networks, proxies, or
    CDN edges that sit in front of Tumblr close the connection when they see
    this, which aiohttp surfaces as a ServerDisconnectedError.

    Leaving ALPN unset makes the remote fall back to plain HTTP/1.1 over TLS,
    which is exactly what aiohttp would use regardless.
    """
    context = ssl.create_default_context()
    context.set_alpn_protocols([])

    return context


def create_connector() -> aiohttp.TCPConnector:
    """Creates an aiohttp connector configured with openblur's SSL context"""
    return aiohttp.TCPConnector(ssl=create_ssl_context())
