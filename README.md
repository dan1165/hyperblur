# hyperblur

Alternative Tumblr frontend. No account, no tracking, no caching. One Go binary,
every asset inside.

## Run

Docker:

```bash
docker compose up -d
```

Pulls `ghcr.io/dan1165/hyperblur:latest`, serves <http://localhost:8000>. Or:

```bash
docker run --rm -p 8000:8000 ghcr.io/dan1165/hyperblur:latest
```

Manual (Go 1.24+):

```bash
go build -o hyperblur ./cmd/hyperblur
./hyperblur
```

Listens on `:8000`. `HYPERBLUR_PORT` changes it. Assets embedded, so binary runs
from anywhere.

## Own Tumblr token (optional)

Default token ships in. Set `HYPERBLUR_TUMBLR_API_TOKEN` to use own. Opens blogs
that need login.

Docker, in `docker-compose.yml`:

```yaml
environment:
  HYPERBLUR_TUMBLR_API_TOKEN: "your-token-here"
```

Manual:

```bash
HYPERBLUR_TUMBLR_API_TOKEN=your-token-here ./hyperblur
```

## What you get

- **Browse:** blogs, tags, search, explore. No account.
- **Explore:** trending, "today on Tumblr", per-type feeds (text, photos, gifs,
  quotes, chats, audio, video, asks).
- **Search:** popular/latest, time-range and post-type filters, blog search, blog
  tag browsing.
- **Posts:** full NPF render — inline text, headings, quotes, chat, lists, images,
  link cards, audio, video, polls with live results, reblog trails, asks.
- **Notes:** replies, reblogs (filterable), likes, paging.
- **Media:** alt text, one-click download (proxied). NSFW shown unblurred.
- **UI:** dark monochrome, system font, no external CDNs, posts always expanded,
  numbered pagination. Browsing needs no JavaScript; small JS adds live poll
  results.

Media loads straight from Tumblr CDN, so Tumblr sees viewer IP.

## Performance

One Go binary on standard `net/http`:

- Shared HTTP transport, 100-idle-per-host pool. Concurrent Tumblr requests reuse
  TLS.
- Media loads straight from Tumblr CDN, no proxy round-trip. Downloads proxied.
- HTML and embedded assets gzipped. Assets carry ETag, cache a week.
- API responses streamed-decoded, fetched fresh every time. No response cache.

## Image

`.github/workflows/docker.yml` builds and pushes to GitHub Container Registry on
every `master` push and `v*` tag:

```bash
docker pull ghcr.io/dan1165/hyperblur:latest
```

## Errors

Unexpected errors log server-side with request and stack trace. Error page shows
same details plus issue link: <https://github.com/dan1165/hyperblur/issues>.

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
