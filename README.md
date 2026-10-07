# openblur

An alternative frontend to Tumblr. No account, no JavaScript, no tracking, no
caching — and no configuration beyond an optional `.env` for a custom Tumblr
token.

## Features

- Browse blogs, tags, search, and explore timelines without an account.
- No JavaScript. Fully monochrome dark UI.
- A download button on every image and video.
- Numbered pagination on blog pages.

> Media loads directly from Tumblr's CDN, so Tumblr can see the IP of anyone
> viewing it. Downloads are proxied through openblur.

## Run

### Docker

```bash
docker compose -f docker-compose.dev.yml up -d --build
```

To run the published image instead: `docker compose up -d`.

### Manual (Python 3.11+)

```bash
python -m venv venv
source venv/bin/activate

pip install -r requirements.txt
python -m src.server
```

## Custom Tumblr token (optional)

openblur ships a default Tumblr API token. To use your own (for example, to
open blogs that require logging in), put it in `.env`:

```bash
cp .env.example .env
```

| Variable | Purpose |
| --- | --- |
| `OPENBLUR_TUMBLR_API_TOKEN` | Custom Tumblr API token. Empty uses the bundled default. |

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
