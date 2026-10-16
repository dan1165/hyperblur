# hyperblur

Alternative Tumblr frontend. No account, no tracking, no cookies. One Go binary,
all assets inside.

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

Listens on `:8000`. `HYPERBLUR_PORT` changes port. Assets embedded: binary runs
from anywhere.

## Configure

`docker compose` reads `.env` on its own. Edit that, or set same variables by
hand.

| Variable | Default | Meaning |
|---|---|---|
| `HYPERBLUR_PORT` | `8000` | Listen port. |
| `HYPERBLUR_PROXY_MEDIA` | `true` | `true` proxies all media through hyperblur: zero CDN, Tumblr never sees viewer IP. `false` loads media straight from Tumblr's CDN: faster, lighter on your server, viewer IP exposed. |
| `HYPERBLUR_TUMBLR_API_TOKEN` | bundled | Own Tumblr API token, to open blogs that require login. |
| `HYPERBLUR_CACHE` | `true` | In-memory page cache. Off = always-fresh responses. |
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
- **UI:** austere monochrome. One neutral ink ramp, hairline rules instead of
  cards, system font, no external CDNs, posts always expanded, numbered
  pagination. Browsing needs no JavaScript; small JS adds live poll results and
  offline mode.

### Install as an app (PWA)

hyperblur is an installable progressive web app. Open it, use browser's
"Install" / "Add to Home Screen". Service worker caches shell and assets: repeat
loads are instant, last pages work offline.

Media goes through hyperblur by default, so browser never contacts Tumblr's CDN
and Tumblr never sees viewer's IP. Set `HYPERBLUR_PROXY_MEDIA=false` to load it
straight from the CDN instead. One-click downloads always go through hyperblur:
they work in either mode and keep the original filename.

## Performance

One Go binary on standard `net/http`. No web framework, no CDN, four
dependencies:

- Shared HTTP transport: HTTP/1.1 + HTTP/2 with idle pings (standard library,
  no `golang.org/x/net`), 100-idle-per-host pool, larger socket buffers.
  Concurrent Tumblr requests reuse TLS.
- Media proxied through hyperblur by default (zero CDN), streamed through a
  pooled copy buffer, cached in the browser. `Range` and conditional headers
  pass through, so video seeking and revalidation work. Downloads add a
  filename. `HYPERBLUR_PROXY_MEDIA=false` loads media straight from Tumblr.
- HTML gzipped once per cached page, at a compression level chosen by
  measurement: level 6 costs ~0.4 ms more than `BestSpeed` on a 535 KB page and
  ships ~10 % fewer bytes (98 KB vs 109 KB). Encoding negotiated per client from
  one cache entry, so gzip and non-gzip visitors share a fill.
- Stylesheets inlined and minified once at startup: page renders with no
  stylesheet request. Other assets carry ETag, cache a week.
- **Only the first few images load eagerly.** Rest are
  `loading="lazy" decoding="async"`, so a 20-post feed opens without fetching 70
  images: browser, and hyperblur's proxy, touch media only as it scrolls in.
  Measured on a photo feed, eager images went from 71 to 3.
- **Off-screen posts skip layout and paint** (`content-visibility: auto` with a
  contained intrinsic size): keeps a 500 KB feed responsive while scrolling.
- No layout shift. Media boxes carry aspect ratio, counts and dates use tabular
  numerals, chrome never reflows. Transitions are 90–140 ms on composited
  properties only, disabled under `prefers-reduced-motion: reduce`.
- **In-memory page cache** with TTL and **stale-while-revalidate**: warm pages
  served from memory; stale pages served immediately while a background goroutine
  refreshes them. **Single-flight** collapses concurrent misses of the same page
  into one upstream request. Two entry points prewarmed at startup. Set
  `HYPERBLUR_CACHE=false` to disable.
- **Faster libraries**: `klauspost/compress` gzip with pooled writers,
  `goccy/go-json`, expirable LRU (`hashicorp/golang-lru`). Stylesheets, asset
  gzip and ETags computed once at startup; render buffers pooled; poll results
  cached for 18 s and single-flighted.
- `GET /healthz` reports status, version, cache size. Build version shown in page
  footer. `hyperblur -healthcheck` probes it: that is what the compose
  healthcheck uses (image is `scratch`, no shell).

## Development

```bash
go vet ./...
go test -race ./...
```

Tests cover URL rewriting and download links, media-path and poll-reference
validation, cache keys and encoding negotiation, NPF formatting, reblog-note
parser. Plus an integration suite that drives the whole handler stack against a
fake Tumblr upstream. CI runs vet and race-enabled tests before building image.

## Image

`.github/workflows/docker.yml` builds and pushes to GitHub Container Registry on
every `master` push and `v*` tag:

```bash
docker pull ghcr.io/dan1165/hyperblur:latest
```

## Errors

Unexpected errors log server-side with request and stack trace. Error page shows
same details plus issue link: <https://github.com/dan1165/hyperblur/issues>.

## Built with

hyperblur is built with the help of AI coding agents, both running DeepSeek:

- [opencode](https://opencode.ai)
- [Claude Code](https://www.anthropic.com/claude-code) (Anthropic)

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
