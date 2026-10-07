# openblur

An alternative frontend to Tumblr. No account, no JavaScript, no tracking, no
caching.

## Features

- Browse blogs, tags, search, and explore timelines without an account.
- No JavaScript. Fully monochrome dark UI.
- A download button on every image and video.

> Media loads directly from Tumblr's CDN, so Tumblr can see the IP of anyone
> viewing it. Downloads are proxied through openblur.

## Run

### Docker

```bash
cp config.example.toml config.toml   # optional
docker compose -f docker-compose.dev.yml up -d --build
```

To run the published image instead: `docker compose up -d`.

### Manual (Python 3.11+)

```bash
python -m venv venv
source venv/bin/activate

pip install -r requirements.txt
cp config.example.toml config.toml   # optional
python -m src.server
```

## Configure

openblur reads `config.toml` (see [`config.example.toml`](./config.example.toml)).
Override the path with `OPENBLUR_CONFIG_LOCATION`.

Optional secrets go in `.env` (see [`.env.example`](./.env.example)):

| Variable | Purpose |
| --- | --- |
| `OPENBLUR_TUMBLR_API_TOKEN` | Custom Tumblr API token; lets openblur open blogs that require logging in. Empty uses the bundled default. |

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
