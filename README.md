<div align="center">
  <h1> openblur </h1>
  <h3> An alternative frontend to Tumblr </h3>
</div>

openblur is a proxy for Tumblr. It browses blogs, tags, search, and the explore
timelines without an account, without JavaScript, and without the tracking.

## Features

- No account or login required.
- No JavaScript.
- Fully monochrome, dark interface.
- A download button on every image and video.
- Media loads directly from Tumblr's CDN for fast pages.
- No caching: every request is fetched fresh from Tumblr.

> **Note:** media (images, audio, video) is served directly from Tumblr's CDN,
> so Tumblr can see the IP address of anyone viewing media through your
> instance. Forced downloads are proxied through openblur.

## Requirements

- Docker, **or** Python 3.11+

## Install

### Docker

Build and run from source:

```bash
git clone <repository-url>
cd openblur

cp config.example.toml config.toml   # optional, the defaults are fine
cp .env.example .env                 # optional, for a custom Tumblr token

docker compose -f docker-compose.dev.yml up -d --build
```

To run the published image instead, use `docker-compose.yml`:

```bash
docker compose up -d
```

openblur is then available at http://localhost:8000.

### Manual

```bash
git clone <repository-url>
cd openblur

python -m venv venv
source venv/bin/activate

pip install -r requirements.txt

cp config.example.toml config.toml   # optional
python -m src.server
```

`python -m src.server` runs Sanic in fast mode: one worker per CPU core and no
access logs. You can also run it through Sanic's CLI (`OPENBLUR_` replaces
`SANIC_` for environment variables):

```bash
sanic src.server.app --host 0.0.0.0 --worker 4
```

## Configure

openblur reads `config.toml` from the current directory. Point elsewhere with
the `OPENBLUR_CONFIG_LOCATION` environment variable. Every value is optional;
see [`config.example.toml`](./config.example.toml).

### Secrets (`.env`)

Copy `.env.example` to `.env`. Real environment variables always take
precedence.

| Variable | Description |
| --- | --- |
| `OPENBLUR_TUMBLR_API_TOKEN` | A custom Tumblr API bearer token. A token tied to a logged-in account lets openblur open blogs that require logging in. Leave empty to use the bundled default. |

Both compose files load `.env` automatically when present.

## Update

```bash
git pull

# Docker
docker compose -f docker-compose.dev.yml up -d --build

# Manual
pip install -r requirements.txt && restart openblur
```

## Instances

[A list of public instances can be found here.](./instances.md)

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
