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

## Configure

`docker compose` reads `.env` automatically. Edit it, or set the same variables
by hand.

| Variable | Default | Meaning |
|---|---|---|
| `HYPERBLUR_PORT` | `8000` | Listen port. |
| `HYPERBLUR_PROXY_MEDIA` | `true` | `true` proxies all media through hyperblur — zero CDN, Tumblr never sees the viewer IP. `false` loads media straight from Tumblr's CDN — faster, lighter on your server, viewer IP exposed. |
| `HYPERBLUR_TUMBLR_API_TOKEN` | bundled | Your own Tumblr API token, to open blogs that require login. |
| `HYPERBLUR_CACHE` | `true` | In-memory page cache. Toggle off for always-fresh responses. |
| `HYPERBLUR_CACHE_TTL` | `60` | Seconds a cached page is served without revalidating. |
| `HYPERBLUR_CACHE_STALE` | `600` | Extra seconds a stale page may be served while it refreshes in the background. |

Docker, in `.env`:

```ini
HYPERBLUR_PROXY_MEDIA=false
HYPERBLUR_TUMBLR_API_TOKEN=your-token-here
```

Manual:

```bash
HYPERBLUR_PROXY_MEDIA=false HYPERBLUR_TUMBLR_API_TOKEN=your-token-here ./hyperblur
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
  results and offline mode.

### Install as an app (PWA)

hyperblur is an installable progressive web app. Open it and use your browser's
"Install" / "Add to Home Screen". A service worker caches the shell and assets,
so repeat loads are instant and the last pages work offline.

Media is proxied through hyperblur by default, so the browser never contacts
Tumblr's CDN and Tumblr never sees the viewer's IP. Set
`HYPERBLUR_PROXY_MEDIA=false` to load it straight from the CDN instead.

## Performance

One Go binary on standard `net/http`:

- Shared HTTP transport, 100-idle-per-host pool. Concurrent Tumblr requests reuse
  TLS.
- Media is proxied through hyperblur by default (zero CDN), streamed and cached
  in the browser; downloads add a filename. `HYPERBLUR_PROXY_MEDIA=false` skips
  the proxy and loads media straight from Tumblr.
- HTML and embedded assets gzipped. Stylesheets inlined, so a page renders with
  no stylesheet request. Other assets carry ETag, cache a week.
- **In-memory page cache** with TTL and **stale-while-revalidate**: warm pages
  are served from memory; stale pages are served immediately while a background
  goroutine refreshes them. **Single-flight** collapses concurrent misses of the
  same page into one upstream request. Set `HYPERBLUR_CACHE=false` to disable.
- **Faster libraries**: `klauspost/compress` gzip, `goccy/go-json`, and an
  expirable LRU (`hashicorp/golang-lru`). The inline stylesheet is precomputed
  once at startup instead of per render.
- `GET /healthz` reports status, version and cache size; the build version is
  shown in the page footer.

## Image

`.github/workflows/docker.yml` builds and pushes to GitHub Container Registry on
every `master` push and `v*` tag:

```bash
docker pull ghcr.io/dan1165/hyperblur:latest
```

## Errors

Unexpected errors log server-side with request and stack trace. Error page shows
same details plus issue link: <https://github.com/dan1165/hyperblur/issues>.

## Disclaimer

Built with the assistance of [opencode](https://opencode.ai), running the latest
DeepSeek Flash model.

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
