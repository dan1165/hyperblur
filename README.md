# openblur

An alternative frontend to Tumblr. No account, no JavaScript, no tracking, no
caching — and no configuration beyond an optional Tumblr token.

## Features

- Browse blogs, tags, search, and explore timelines without an account.
- No JavaScript. Fully monochrome dark UI, with posts expanded by default.
- A download button on every image and video, plus a one-click copy-link.
- Numbered pagination on blog pages.

> Media loads directly from Tumblr's CDN, so Tumblr can see the IP of anyone
> viewing it. Downloads are proxied through openblur.

## Run

### Docker

```bash
docker compose up -d --build
```

### Manual (Python 3.11+)

```bash
python -m venv venv
source venv/bin/activate

pip install -r requirements.txt
python -m server
```

## Custom Tumblr token (optional)

openblur ships a default Tumblr API token. To use your own — for example, to
open blogs that require logging in — set `OPENBLUR_TUMBLR_API_TOKEN`.

With Docker, edit it in `docker-compose.yml`:

```yaml
environment:
  OPENBLUR_TUMBLR_API_TOKEN: "your-token-here"
```

Manually:

```bash
OPENBLUR_TUMBLR_API_TOKEN=your-token-here python -m server
```

## Performance

Runs on Sanic with uvloop and httptools, one worker per CPU core, keep-alive
and pooled connections, and orjson. Every request is fetched fresh from Tumblr
(no caching).

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
