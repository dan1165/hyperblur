<div align="center"> 
  <h1> Priviblur </h1>
  <div>
  <a href="https://www.gnu.org/licenses/agpl-3.0.en.html">
    <img alt="License: AGPLv3" src="https://shields.io/badge/License-AGPL%20v3-blue.svg">
  </a>
  <a href="https://github.com/syeopite/priviblur/commits/master">
    <img alt="GitHub commits" src="https://img.shields.io/github/commit-activity/y/syeopite/priviblur?color=e69419&label=commits">
  </a>
  </div>
  <blockquote> <h4> Inspired by projects like <a href="https://github.com/iv-org/invidious"> Invidious</a></h4> </blockquote>
  <h3> Priviblur is an alternative frontend to Tumblr with a touch of modern design </h3>
</div>

<br/>

![Example output](./screenshots/example.png)

Priviblur is a proxy. It makes requests to Tumblr in lieu of you allowing you browse without being tracked. 

It has no account requirement either. Allowing you to view your favorite blogs without ever needing to login.

It is lightweight and works without Javascript. Allowing for a much faster experience compared with Tumblr.

It has a modern design. Although perhaps not quite there yet, the project aims to replicate the experience seen on modern software.

It is licensed under the AGPLv3 ensuring that itself and all instances are free and open. Forever. 

## Features

- Browse blogs, tags, search, and Tumblr's explore timelines without an account.
- Works without Javascript.
- Always-on dark theme.
- English interface.
- A download button on every image and video.
- Media is loaded directly from Tumblr's CDN for fast page loads.

> [!NOTE]
> Media (images, audio, and video) is served directly from Tumblr's CDN. This keeps pages fast and light on the server, but it means Tumblr can see the IP address of anyone viewing media through your instance. Forced downloads (the download buttons) are proxied through Priviblur.

## Instances

[A list of public instances can be found here.](./instances.md)

Priviblur has no official instance

## Installation

### Docker

You can install Priviblur through the official docker images here: https://quay.io/repository/syeopite/priviblur

A compose file to use this image is provided in the repository. To build the image from source, use `docker-compose.dev.yml` instead.

Configuration is then done by creating/editing a `config.toml` based off the example config, and optionally a `.env` file for secrets. See the configuration section below.

> [!TIP]
> Priviblur **officially** only provide images of each stable release. For an image built off of master you can use the image provided by PussTheCat.org here: https://github.com/PussTheCat-org/docker-priviblur-quay

### Manual

Priviblur requires Python 3.11 or newer.

```bash

git clone "https://github.com/syeopite/priviblur"
cd priviblur 

python -m venv venv 
source venv/bin/activate

pip install -r requirements.txt

pybabel compile -d locales -D priviblur

python -m src.server

# You can also launch Priviblur through Sanic (our web framework)'s CLI tool
# Prefix any environmental variables with PRIVIBLUR_ instead of SANIC_
# For more information see https://sanic.dev/en/guide/deployment/running.html and related pages
sanic src.server.app  --host 0.0.0.0  --worker <WORKERS>
```

When launched with `python -m src.server`, Priviblur runs Sanic in its `fast` mode, which automatically spawns one worker per available CPU core and disables access logs.

## Configuration

[Example config provided here](./config.example.toml)

Priviblur reads its configuration from `config.toml`. The location can be overridden with the `PRIVIBLUR_CONFIG_LOCATION` environment variable.

```bash
cp config.example.toml config.toml
```

### Optional secrets (`.env`)

Priviblur also reads a `.env` file (if present) for optional secrets. Real environment variables always take precedence, and every value is optional.

```bash
cp .env.example .env
```

| Variable | Description |
| --- | --- |
| `PRIVIBLUR_TUMBLR_API_TOKEN` | A custom Tumblr API bearer token. Supplying a token tied to a logged-in Tumblr account lets Priviblur access blogs that require logging in. Leave empty to use Priviblur's bundled default token. |

Both `docker-compose.yml` and `docker-compose.dev.yml` load `.env` automatically when it exists.

## Donate 

Bitcoin (BTC): bc1qas50t647kclvhv4ra4h0zw0kmh8wd9jqt6yk9u

Monero (XMR): 89bn95bwPaqD3t7EDjAKKnYJYPRZDLBym8DV1Spyc4LEaVezknHur7DG5iXxg1e7Jwcr9v8MSFmvj6VvDNARwTcUBeSfB2P
