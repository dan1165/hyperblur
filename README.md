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

### Manual (Go 1.24+)

```bash
go build -o openblur ./cmd/openblur
./openblur
```

openblur listens on `:8000`; set `OPENBLUR_PORT` to change it. The stylesheets,
scripts, fonts and images are embedded in the binary, so it runs from anywhere.

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
OPENBLUR_TUMBLR_API_TOKEN=your-token-here ./openblur
```

## Performance

A single Go binary on the standard `net/http` server, with pooled keep-alive
connections. Every request is fetched fresh from Tumblr (no caching).

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
