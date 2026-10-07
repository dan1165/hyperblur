<div align="center"> 
  <h1> openblur </h1>
  <h3> An alternative frontend to Tumblr with a touch of modern design </h3>
</div>

<br/>

openblur is a proxy. It makes requests to Tumblr in lieu of you allowing you browse without being tracked. 

It has no account requirement either. Allowing you to view your favorite blogs without ever needing to login.

It is lightweight and works without Javascript. Allowing for a much faster experience compared with Tumblr.

It has a modern design. Although perhaps not quite there yet, the project aims to replicate the experience seen on modern software.

It is licensed under the AGPLv3 ensuring that itself and all instances are free and open. Forever. 

## Features

- Browse blogs, tags, search, and Tumblr's explore timelines without an account.
- Works without Javascript.
- Fully monochrome dark interface with sharp corners.
- English interface.
- A download button on every image and video.
- Media is loaded directly from Tumblr's CDN for fast page loads.
- No caching: every request is fetched fresh from Tumblr.

> [!NOTE]
> Media (images, audio, and video) is served directly from Tumblr's CDN. This keeps pages fast and light on the server, but it means Tumblr can see the IP address of anyone viewing media through your instance. Forced downloads (the download buttons) are proxied through openblur.

## Instances

[A list of public instances can be found here.](./instances.md)

openblur has no official instance

## Installation

### Docker

A compose file to run the published image is provided in `docker-compose.yml`. To build the image from source, use `docker-compose.dev.yml` instead.

Configuration is then done by creating/editing a `config.toml` based off the example config, and optionally a `.env` file for secrets. See the configuration section below.

### Manual

openblur requires Python 3.11 or newer.

```bash

git clone <repository-url>
cd openblur 

python -m venv venv 
source venv/bin/activate

pip install -r requirements.txt

pybabel compile -d locales -D openblur

python -m src.server

# You can also launch openblur through Sanic (our web framework)'s CLI tool
# Prefix any environmental variables with OPENBLUR_ instead of SANIC_
sanic src.server.app  --host 0.0.0.0  --worker <WORKERS>
```

When launched with `python -m src.server`, openblur runs Sanic in its `fast` mode, which automatically spawns one worker per available CPU core and disables access logs.

## Configuration

[Example config provided here](./config.example.toml)

openblur reads its configuration from `config.toml`. The location can be overridden with the `OPENBLUR_CONFIG_LOCATION` environment variable.

```bash
cp config.example.toml config.toml
```

### Optional secrets (`.env`)

openblur also reads a `.env` file (if present) for optional secrets. Real environment variables always take precedence, and every value is optional.

```bash
cp .env.example .env
```

| Variable | Description |
| --- | --- |
| `OPENBLUR_TUMBLR_API_TOKEN` | A custom Tumblr API bearer token. Supplying a token tied to a logged-in Tumblr account lets openblur access blogs that require logging in. Leave empty to use openblur's bundled default token. |

Both `docker-compose.yml` and `docker-compose.dev.yml` load `.env` automatically when it exists.
