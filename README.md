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
| `HYPERBLUR_TUMBLR_API_TOKEN` | bundled | Own Tumblr API token, to open blogs that require login. |

Docker, in `.env`:

```ini
HYPERBLUR_TUMBLR_API_TOKEN=your-token-here
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
- **UI:** austere monochrome. One neutral ink ramp, hairline rules instead of
  cards, system font, no external CDNs, posts always expanded, numbered
  pagination. Browsing needs no JavaScript; small JS adds live poll results and
  offline mode.

### Install as an app (PWA)

hyperblur is an installable progressive web app. Open it, use browser's
"Install" / "Add to Home Screen". Service worker caches shell and assets: repeat
loads are instant, last pages work offline.

Display media loads straight from Tumblr's CDN: fast, and light on your server.
One-click downloads come through hyperblur instead, so the file is saved with its
real name rather than opening in a tab.

## Performance

One Go binary on standard `net/http`. No web framework, no CDN, four
dependencies:

- Shared HTTP transport: HTTP/1.1 + HTTP/2 with idle pings (standard library,
  no `golang.org/x/net`), 100-idle-per-host pool, larger socket buffers.
  Concurrent Tumblr requests reuse TLS.
- Display media is served by Tumblr's CDN directly: the browser fetches images
  and video from the edge, not from your server. Downloads take the one route
  that does pass through hyperblur (`/download/...`), streamed through a pooled
  copy buffer with a `Content-Disposition` filename.
- HTML is gzipped per response with a pooled writer, at a compression level
  chosen by measurement: level 6 costs ~0.4 ms more than `BestSpeed` on a 535 KB
  page and ships ~10 % fewer bytes (98 KB vs 109 KB).
- Stylesheets inlined and minified once at startup: page renders with no
  stylesheet request. Other assets carry ETag, cache a week.
- **Only the first few images load eagerly.** Rest are
  `loading="lazy" decoding="async"`, so a 20-post feed opens without fetching 70
  images: the browser touches media only as it scrolls in. Measured on a photo
  feed, eager images went from 71 to 3.
- **Off-screen posts skip layout and paint** (`content-visibility: auto` with a
  contained intrinsic size): keeps a 500 KB feed responsive while scrolling.
- No layout shift. Media boxes carry aspect ratio, counts and dates use tabular
  numerals, chrome never reflows. Transitions are 90–140 ms on composited
  properties only, disabled under `prefers-reduced-motion: reduce`.
- **No page cache, no upstream fan-out.** One view fetches its page from Tumblr
  once; that is the whole request path. Poll results are cached for 18 s in a
  512-entry TTL map, so a post full of polls does not re-fetch while a reader
  watches them.
- **Standard library only.** No third-party module in `go.mod`: gzip, JSON,
  HTTP/2, hashing and the embed all come from Go itself. Stylesheets, asset gzip
  and ETags are computed once at startup; render buffers are pooled.
- `GET /healthz` reports status and version. Build version shown in page footer.
  `hyperblur -healthcheck` probes it: that is what the compose healthcheck uses
  (image is `scratch`, no shell).

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
