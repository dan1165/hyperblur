"""Generate npf/testdata/golden.json from the reference npf_renderer.

Run with the venv that has npf_renderer==0.13.0 installed:
    /tmp/opencode/npfvenv/bin/python npf/testdata/gen_golden.py

It writes a JSON file of {name: {content, layout, opts, error, html}} used by
npf/golden_test.go. The Go port is expected to reproduce `html` and `error`
for every case.
"""
import json
import os
import urllib.parse

import npf_renderer

CASES = {}

def add(name, content, layout=None, *, url_prefix=None, forbid_iframes=False, truncate=False):
    url_handler = None
    if url_prefix:
        url_handler = lambda u: url_prefix + u
    has_err, html = npf_renderer.format_npf(
        content, layout,
        url_handler=url_handler,
        forbid_external_iframes=forbid_iframes,
        pretty_html=False,
        truncate=truncate,
    )
    CASES[name] = {
        "content": content,
        "layout": layout,
        "opts": {"url_prefix": url_prefix, "forbid_iframes": forbid_iframes, "truncate": truncate},
        "error": bool(has_err),
        "html": html,
    }

IMG = {
    "url": "https://64.media.tumblr.com/abc/s540x810/1.jpg",
    "type": "image/jpeg", "width": 540, "height": 810,
    "has_original_dimensions": True,
}

# --- text ---
add("text_simple", [{"type": "text", "text": "hello world"}])
add("text_empty", [{"type": "text", "text": ""}])
add("text_heading1", [{"type": "text", "text": "Title", "subtype": "heading1"}])
add("text_heading2", [{"type": "text", "text": "Sub", "subtype": "heading2"}])
add("text_quote", [{"type": "text", "text": "To be", "subtype": "quote"}])
add("text_quirky", [{"type": "text", "text": "wow", "subtype": "quirky"}])
add("text_chat", [{"type": "text", "text": "Me: hi", "subtype": "chat"}])
add("text_indented", [{"type": "text", "text": "quoted", "subtype": "indented"}])
add("text_indent_nested", [
    {"type": "text", "text": "outer", "subtype": "indented"},
    {"type": "text", "text": "inner", "subtype": "indented", "indent_level": 1},
])
add("text_ordered_list", [
    {"type": "text", "text": "one", "subtype": "ordered-list-item"},
    {"type": "text", "text": "two", "subtype": "ordered-list-item"},
])
add("text_unordered_list", [
    {"type": "text", "text": "a", "subtype": "unordered-list-item"},
    {"type": "text", "text": "b", "subtype": "unordered-list-item"},
])
add("text_list_nested", [
    {"type": "text", "text": "a", "subtype": "unordered-list-item"},
    {"type": "text", "text": "a1", "subtype": "unordered-list-item", "indent_level": 1},
    {"type": "text", "text": "b", "subtype": "unordered-list-item"},
])

# --- inline ---
add("inline_small", [{"type": "text", "text": "some small text",
                      "formatting": [{"start": 5, "end": 10, "type": "small"}]}])
add("inline_bold_italic", [{"type": "text", "text": "a bold italic end",
                            "formatting": [{"start": 2, "end": 6, "type": "bold"},
                                           {"start": 7, "end": 13, "type": "italic"}]}])
add("inline_strike", [{"type": "text", "text": "strike me",
                       "formatting": [{"start": 0, "end": 6, "type": "strikethrough"}]}])
add("inline_link", [{"type": "text", "text": "Found this link for you",
                     "formatting": [{"start": 6, "end": 10, "type": "link", "url": "https://www.nasa.gov"}]}])
add("inline_mention", [{"type": "text", "text": "Shout out to @david",
                        "formatting": [{"start": 13, "end": 19, "type": "mention",
                                        "blog": {"uuid": "t:1", "name": "david", "url": "https://davidslog.com/"}}]}])
add("inline_color", [{"type": "text", "text": "Celebrate Pride Month",
                      "formatting": [{"start": 10, "end": 15, "type": "color", "hex": "#ff492f"}]}])
add("inline_overlap", [{"type": "text", "text": "abcdefghij",
                        "formatting": [{"start": 1, "end": 5, "type": "bold"},
                                       {"start": 3, "end": 7, "type": "italic"}]}])
add("inline_nested_same", [{"type": "text", "text": "abcdefghij",
                            "formatting": [{"start": 0, "end": 10, "type": "bold"},
                                           {"start": 2, "end": 5, "type": "italic"}]}])
add("inline_back_to_back", [{"type": "text", "text": "abcdef",
                             "formatting": [{"start": 0, "end": 2, "type": "bold"},
                                            {"start": 2, "end": 4, "type": "italic"}]}])
add("inline_out_of_bounds", [{"type": "text", "text": "abc",
                              "formatting": [{"start": 0, "end": 100, "type": "bold"}]}])
add("inline_escaped_text", [{"type": "text", "text": "a < b & c > d \"q\""}])
add("inline_escaped_attr", [{"type": "text", "text": "x",
                             "formatting": [{"start": 0, "end": 1, "type": "color", "hex": "#fff\"onx"}]}])
add("inline_same_start", [{"type": "text", "text": "abcdefg",
                           "formatting": [{"start": 0, "end": 3, "type": "bold"},
                                          {"start": 0, "end": 5, "type": "italic"}]}])

# --- image ---
add("image_basic", [{"type": "image", "media": [IMG]}])
add("image_alt", [{"type": "image", "media": [IMG], "alt_text": "a cat"}])
add("image_caption", [{"type": "image", "media": [IMG], "caption": "look"}])
add("image_cropped_skip", [{"type": "image", "media": [
    dict(IMG, cropped=True, url="https://64.media.tumblr.com/abc/s540x810/crop.jpg"),
    dict(IMG, url="https://64.media.tumblr.com/abc/s540x810/orig.jpg"),
]}])
add("image_attribution_link", [{"type": "image", "media": [IMG],
                               "attribution": {"type": "link", "url": "https://example.com/a"}}])
add("image_attribution_post", [{"type": "image", "media": [IMG],
                               "attribution": {"type": "post", "url": "https://ex.tumblr.com/1",
                                               "post": {"id": "1"},
                                               "blog": {"uuid": "t:1", "name": "ex", "url": "https://ex.tumblr.com"}}}])
add("image_attribution_blog", [{"type": "image", "media": [IMG],
                               "attribution": {"type": "blog", "url": "https://ex.tumblr.com",
                                               "blog": {"uuid": "t:1", "name": "ex"}}}])
add("image_attribution_app", [{"type": "image", "media": [IMG],
                               "attribution": {"type": "app", "url": "https://app.example",
                                               "app_name": "CoolApp", "display_text": "cool"}}])
add("image_attribution_unknown", [{"type": "image", "media": [IMG],
                                   "attribution": {"type": "wat"}}])

# --- link ---
add("link_basic", [{"type": "link", "url": "https://example.com",
                    "title": "Example", "description": "desc", "author": "bob",
                    "site_name": "example.com"}])
add("link_no_site_name", [{"type": "link", "url": "https://example.com/x", "title": "T"}])
add("link_poster", [{"type": "link", "url": "https://example.com", "title": "T", "poster": [IMG]}])

# --- video ---
VID = {"url": "https://ve.media.tumblr.com/abc/v.mp4", "type": "video/mp4", "width": 640, "height": 360}
POSTER = {"url": "https://64.media.tumblr.com/p.jpg", "width": 640, "height": 360}
add("video_native", [{"type": "video", "provider": "tumblr", "media": [VID], "poster": [POSTER]}])
add("video_embed_iframe", [{"type": "video", "provider": "youtube",
                            "embed_iframe": {"url": "https://www.youtube.com/embed/x", "width": 640, "height": 360}}])
add("video_embed_html", [{"type": "video", "provider": "vimeo", "embed_html": "<iframe src=\"x\"></iframe>"}])
add("video_fallback", [{"type": "video", "url": "https://example.com/v", "provider": "vimeo"}])
add("video_non_tumblr_source", [{"type": "video", "provider": "tumblr",
                                 "media": [dict(VID, url="https://cdn.example.com/v.mp4")]}])

# --- audio ---
AUD = {"url": "https://a.tumblr.com/abc.mp3", "type": "audio/mpeg"}
add("audio_native", [{"type": "audio", "provider": "tumblr", "media": [AUD],
                      "title": "Song", "artist": "Artist", "album": "Album", "poster": [POSTER]}])
add("audio_embed_url", [{"type": "audio", "provider": "spotify", "embed_url": "https://open.spotify.com/embed/x"}])
add("audio_embed_html", [{"type": "audio", "provider": "spotify", "embed_html": "<div>player</div>"}])
add("audio_fallback", [{"type": "audio", "url": "https://example.com/a", "provider": "spotify"}])

# --- poll ---
add("poll_basic", [{"type": "poll", "client_id": "pid", "question": "Q?",
                    "answers": [{"client_id": "a1", "answer_text": "Yes"},
                                {"client_id": "a2", "answer_text": "No"}],
                    "timestamp": 1700000000, "settings": {"expireAfter": 604800}}])
add("poll_camel", [{"type": "poll", "clientId": "pid", "question": "Q?",
                    "answers": [{"clientId": "a1", "answerText": "Yes"},
                                {"clientId": "a2", "answerText": "No"}],
                    "timestamp": 1700000000, "settings": {"expireAfter": 604800}}])

# --- unsupported ---
add("unsupported", [{"type": "text", "text": "hi"}, {"type": "nope", "x": 1}])

# --- layouts ---
add("layout_rows_images", [
    {"type": "image", "media": [dict(IMG, width=540, height=810)]},
    {"type": "image", "media": [dict(IMG, width=540, height=540)]},
    {"type": "text", "text": "after"},
], layout=[{"type": "rows", "display": [{"blocks": [0, 1]}, {"blocks": [2]}]}])
add("layout_truncate", [
    {"type": "text", "text": "first"},
    {"type": "text", "text": "second"},
], layout=[{"type": "rows", "truncateAfter": 0, "display": [{"blocks": [0]}, {"blocks": [1]}]}],
    truncate=True)
add("layout_ask", [
    {"type": "text", "text": "why?"},
    {"type": "text", "text": "because"},
], layout=[{"type": "ask", "blocks": [0], "attribution": {"type": "blog", "url": "https://asker.tumblr.com",
                                                          "blog": {"uuid": "t:2", "name": "asker"}}}])

# --- url handler / iframes ---
add("url_handler", [{"type": "text", "text": "hi",
                     "formatting": [{"start": 0, "end": 2, "type": "link", "url": "https://x.com"}]}],
    url_prefix="PROXIED:")

out = os.path.join(os.path.dirname(os.path.abspath(__file__)), "golden.json")
with open(out, "w") as f:
    json.dump(CASES, f, indent=1, sort_keys=True)

print(f"wrote {len(CASES)} cases to {out}")
