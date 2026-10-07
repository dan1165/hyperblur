# openblur

An alternative frontend to Tumblr. No account, no tracking, no caching — and no
configuration beyond an optional Tumblr token. A single Go binary with every
asset embedded.

## Features

- **Browse without an account:** blogs, tags, search, and explore.
- **Explore:** trending, "today on Tumblr", and per-post-type feeds (text,
  photos, gifs, quotes, chats, audio, video, asks).
- **Search:** popular/latest sorting, time-range filters, and post-type
  filters. Blog search and blog tag browsing too.
- **Full NPF rendering:** text with inline formatting (bold, italics,
  strikethrough, small, colour, links, mentions), headings, quotes, chat and
  lists; images; link cards; audio; video (native + embeds); polls with live
  results.
- **Reblogs:** post trails with per-trail headers and reblog attribution, plus
  asks.
- **Notes viewer:** replies, reblogs (filterable) and likes, with paging.
- **Media:** image alt-text widget, one-click download for images/audio/video,
  and a copy-link button. Downloads are proxied through openblur.
- **NSFW:** community-labelled posts are shown directly, unblurred.
- **UI:** dark monochrome theme, system font (no web fonts or external CDNs),
  posts expanded by default, numbered pagination on blog pages.
- **Works without JavaScript:** navigation and media browsing need none; a
  small amount of JS adds poll results and copy-link.

> Media loads directly from Tumblr's CDN, so Tumblr can see the IP of anyone
> viewing it.

## Run

### Docker

Pulls the published image (`ghcr.io/dan1165/openblur:latest`):

```bash
docker compose up -d
```

Or run it directly:

```bash
docker run --rm -p 8000:8000 ghcr.io/dan1165/openblur:latest
```

### Manual (Go 1.24+)

```bash
go build -o openblur ./cmd/openblur
./openblur
```

openblur listens on `:8000`; set `OPENBLUR_PORT` to change it. The stylesheets,
scripts and images are embedded in the binary, so it runs from anywhere.

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

## Image

Images are built and pushed to the GitHub Container Registry by
`.github/workflows/docker.yml` on every push to `master` (and on `v*` tags):

```bash
docker pull ghcr.io/dan1165/openblur:latest
```

## Errors

Unexpected errors are logged server-side with the request and a stack trace,
and the error page shows the same details plus a link to open an issue at
<https://github.com/dan1165/openblur/issues>.

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
