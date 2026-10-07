"""Generate render/testdata/openblur_golden.json from openblur's Python renderer.

Run with a venv that has openblur's requirements installed:
    /tmp/opencode/npfvenv/bin/python render/testdata/gen_openblur_golden.py
"""
import asyncio
import json
import os
import sys

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
sys.path.insert(0, REPO)

import helpers.ext_npf_renderer as ext
import i18n


class Preferences:
    expand_posts = True


class Ctx:
    preferences = Preferences()
    language = "en_US"


class AppCtx:
    LANGUAGES = i18n.initialize_locales()
    OPENBLUR_PARENT_DIR_PATH = REPO

    @staticmethod
    def create_user_friendly_error_message(request, exception):
        return (exception.__class__.__qualname__, str(exception), "")


class App:
    ctx = AppCtx()


class Request:
    ctx = Ctx()
    app = App()


REQ = Request()
CASES = {}


def add(name, content, layout=None, blog_name=None, post_id=None):
    render_error, html = asyncio.run(
        ext.format_npf(content, layout, blog_name=blog_name, post_id=post_id, request=REQ)
    )
    CASES[name] = {
        "content": content,
        "layout": layout,
        "blog_name": blog_name,
        "post_id": post_id,
        "error_name": render_error[0] if render_error else None,
        "error_message": render_error[1] if render_error else None,
        "html": html,
    }


IMG = {
    "url": "https://64.media.tumblr.com/abc/s540x810/1.jpg",
    "type": "image/jpeg", "width": 540, "height": 810,
    "has_original_dimensions": True,
}
VID = {"url": "https://ve.media.tumblr.com/abc/v.mp4", "type": "video/mp4", "width": 640, "height": 360}

add("image_download", [{"type": "image", "media": [IMG]}])
add("image_alt_download", [{"type": "image", "media": [IMG], "alt_text": "a cat"}])
add("image_default_alt_no_widget", [{"type": "image", "media": [IMG], "alt_text": "image"}])
add("video_download", [{"type": "video", "provider": "tumblr", "media": [VID]}])
add("poll_noscript", [{"type": "poll", "client_id": "pid", "question": "Q?",
                       "answers": [{"client_id": "a1", "answer_text": "Yes"},
                                   {"client_id": "a2", "answer_text": "No"}],
                       "timestamp": 1700000000, "settings": {"expireAfter": 604800}}],
    blog_name="blog", post_id="123")
add("poll_no_ids", [{"type": "poll", "client_id": "pid", "question": "Q?",
                     "answers": [{"client_id": "a1", "answer_text": "Yes"}],
                     "timestamp": 1700000000, "settings": {"expireAfter": 604800}}])
add("unsupported_error", [{"type": "text", "text": "hi"}, {"type": "nope"}])
add("url_handler_image", [{"type": "image", "media": [dict(IMG, url="https://49.media.tumblr.com/x.jpg")]},
                          {"type": "link", "url": "https://example.com", "title": "T"}])

# NOTE: a not-yet-expired poll is deliberately not included: its "remaining
# time" string depends on the wall clock and would drift.

out = os.path.join(os.path.dirname(os.path.abspath(__file__)), "openblur_golden.json")
with open(out, "w") as f:
    json.dump(CASES, f, indent=1, sort_keys=True)
print(f"wrote {len(CASES)} cases to {out}")
