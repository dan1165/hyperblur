# hyperblur

Alternative frontend to Tumblr. No account, no tracking, no caching. One Go
binary with every asset embedded.

## Run

Docker:

```bash
docker compose up -d
```

Opens the published image (`ghcr.io/dan1165/hyperblur:latest`) on
<http://localhost:8000>. Or run it directly:

```bash
docker run --rm -p 8000:8000 ghcr.io/dan1165/hyperblur:latest
```

Manual (Go 1.24+):

```bash
go build -o hyperblur ./cmd/hyperblur
./hyperblur
```

Listens on `:8000`; set `HYPERBLUR_PORT` to change it. Stylesheets, scripts and
images are embedded, so the binary runs from anywhere.

## Use your own Tumblr token (optional)

hyperblur ships a default Tumblr API token. Set `HYPERBLUR_TUMBLR_API_TOKEN` to
use your own — for example, to open blogs that require a login.

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

- **Browse:** blogs, tags, search and explore, no account.
- **Explore:** trending, "today on Tumblr", and per-type feeds (text, photos,
  gifs, quotes, chats, audio, video, asks).
- **Search:** popular/latest sorting, time-range and post-type filters, blog
  search, blog tag browsing.
- **Posts:** full NPF rendering — inline text (bold, italics, strikethrough,
  small, colour, links, mentions), headings, quotes, chat, lists, images, link
  cards, audio, video (native + embeds), polls with live results, reblog trails,
  asks.
- **Notes:** replies, reblogs (filterable) and likes, with paging.
- **Media:** alt text and one-click download for images/audio/video; downloads
  are proxied through hyperblur. NSFW posts are shown directly, unblurred.
- **UI:** dark monochrome, system font, no external CDNs, posts always expanded,
  numbered pagination. Browsing needs no JavaScript; a little adds live poll
  results.

Media loads straight from Tumblr's CDN, so Tumblr can see the IP of anyone
viewing it.

## Performance

One Go binary on the standard `net/http` server:

- Shared HTTP transport with a 100-per-host idle pool, so concurrent requests to
  Tumblr reuse TLS connections.
- Media loads straight from Tumblr's CDN — no proxy round-trip; downloads are
  proxied.
- HTML and embedded assets are gzipped; assets carry ETags and cache in the
  browser for a week.
- API responses are streamed-decoded and fetched fresh every time (no response
  caching).

## Image

`.github/workflows/docker.yml` builds and pushes the image to the GitHub
Container Registry on every push to `master` and on `v*` tags:

```bash
docker pull ghcr.io/dan1165/hyperblur:latest
```

## Errors

Unexpected errors are logged server-side with the request and a stack trace. The
error page shows the same details plus a link to open an issue at
<https://github.com/dan1165/hyperblur/issues>.

## License

AGPLv3 — see [`LICENSE`](./LICENSE).
