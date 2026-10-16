# hyperblur

Alternative Tumblr frontend. No account, no tracking, no cookies. One Go binary,
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

Manual (Go 1.27+):

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
`HYPERBLUR_PROXY_MEDIA=false` to load it straight from the CDN instead. One-click
downloads always go through hyperblur, so they work in either mode and keep the
original filename.

## Performance

One Go binary on standard `net/http` — no web framework, no CDN, four
dependencies:

- Shared HTTP transport: HTTP/1.1 + HTTP/2 with idle pings (standard library,
  no `golang.org/x/net`), 100-idle-per-host pool, larger socket buffers.
  Concurrent Tumblr requests reuse TLS.
- Media is proxied through hyperblur by default (zero CDN), streamed through a
  pooled copy buffer and cached in the browser; `Range` and conditional headers
  pass through, so video seeking and revalidation work. Downloads add a
  filename. `HYPERBLUR_PROXY_MEDIA=false` loads media straight from Tumblr.
- HTML is gzipped once per cached page, at a compression level chosen by
  measurement: level 6 costs ~0.4 ms more than `BestSpeed` on a 535 KB page and
  ships ~10 % fewer bytes (98 KB vs 109 KB). Encoding is negotiated per client
  from one cache entry, so gzip and non-gzip visitors share a fill.
- Stylesheets inlined, so a page renders with no stylesheet request. Other
  assets carry ETag and cache a week.
- **In-memory page cache** with TTL and **stale-while-revalidate**: warm pages
  are served from memory; stale pages are served immediately while a background
  goroutine refreshes them. **Single-flight** collapses concurrent misses of the
  same page into one upstream request, and the two entry points are prewarmed at
  startup. Set `HYPERBLUR_CACHE=false` to disable.
- **Faster libraries**: `klauspost/compress` gzip with pooled writers,
  `goccy/go-json`, and an expirable LRU (`hashicorp/golang-lru`). Stylesheets,
  asset gzip and ETags are computed once at startup; render buffers are pooled;
  poll results are cached for 18 s and single-flighted.
- `GET /healthz` reports status, version and cache size; the build version is
  shown in the page footer. `hyperblur -healthcheck` probes it, which is what the
  compose healthcheck uses (the image is `scratch`, so there is no shell).

## Development

```bash
go vet ./...
go test -race ./...
```

Tests cover URL rewriting and download links, media-path and poll-reference
validation, cache keys and encoding negotiation, NPF formatting, and the
reblog-note parser — plus an integration suite that drives the whole handler
stack against a fake Tumblr upstream. CI runs vet and the race-enabled tests
before building the image.

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
