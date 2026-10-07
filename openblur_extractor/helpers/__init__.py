"""Helpers shared across the openblur extractor"""

import logging
import ssl
from typing import List, Tuple

import aiohttp

LOGGER = logging.getLogger("openblur-extractor")


def dig_dict(target, keys: List | Tuple):
    """Digs through a dictionary. Returns none if a given key is missing"""
    for key in keys:
        if isinstance(target, dict):
            target = target.get(key)
        else:
            return None

    return target


def create_connector() -> aiohttp.TCPConnector:
    """Creates an aiohttp connector for requests to Tumblr

    ALPN is intentionally left unset. aiohttp only speaks HTTP/1.1 and some
    networks, proxies, or CDN edges in front of Tumblr close the connection
    when it advertises "http/1.1" through ALPN.
    """
    context = ssl.create_default_context()
    context.set_alpn_protocols([])

    return aiohttp.TCPConnector(
        ssl=context,
        limit=200,  # concurrent connections
        keepalive_timeout=30,  # reuse TLS connections across requests
        ttl_dns_cache=300,  # cache DNS lookups
    )
