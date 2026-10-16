# AGENTS.md

hyperblur - a Tumblr frontend. One Go binary, assets embedded, no web framework.

## Layout

- `cmd/hyperblur/main.go` - entrypoint: flags, `http.Server`, graceful shutdown.
- `web/` - HTTP layer: routing + media (`server.go`), handlers, views, `cache.go`, `version.go`.
- `web/assets/` - CSS/JS/PWA files, embedded with `//go:embed assets`. Put assets here, not at the repo root.
- `tumblr/` - Tumblr API client, models, JSON parsing.
- `npf/` - Neue Post Format -> HTML renderer.

One module (`github.com/dan1165/hyperblur`), Go 1.27 (`go.mod`, Dockerfile and CI must stay in step).

## Commands

```bash
go vet ./...                                  # CI runs this
go test ./...                                 # unit + integration
go test -race ./...                           # CI runs this (needs cgo)
go test ./web/ -run TestPageCache                # one package / one test
go build -o bin/hyperblur ./cmd/hyperblur     # then ./bin/hyperblur  (listens :8000)
```

Build into `bin/` on Windows: `.gitignore` covers `/bin/` and `/hyperblur`, but not `hyperblur.exe`.

`hyperblur -healthcheck` probes local `/healthz` and sets the exit code - it is the
compose healthcheck (the image is `scratch`, with no shell or curl).

## Routing & cache (easy to break)

- ServeMux patterns cover `/assets/`, `/tblr/...`, `/at/{path...}`, `/api/v1/poll/...`,
  `/healthz` and `/{$}`. Everything else (explore/search/tagged/blog) is hand-split in
  `handleFallback` (`GET /`) - blog paths collide with the wildcard patterns, so don't
  move them into the mux.
- `cacheablePath` (`web/cache.go`) decides what the page cache may store: it excludes
  `/assets/`, `/tblr/`, `/api/`, `/at/`, `/robots.txt`, `/manifest.webmanifest`,
  `/sw.js`, `/healthz`. Any other `200` HTML page is cached automatically.
- A cached page is stored as **raw HTML + one precomputed gzip**; the client encoding is
  negotiated at serve time and the cache key is just `path?query`. `writePage`
  deliberately writes uncompressed for cacheable paths so the cache can do the encoding
  - don't re-add gzip there.
- The server intentionally has **no `WriteTimeout`**: media is streamed and a deadline
  would cut long downloads.

## UI layer (austere monochrome)

- Stylesheets: `web/assets/css/{base,post,blog}.css`. `base.css` holds the token
  block (`--ink-*`, spacing, motion, type scale), the shell, nav, controls and
  paging; `post.css` holds posts plus the whole NPF block set; `blog.css` is
  appended on blog pages only. `inlineCSS` (`web/server.go`) concatenates base +
  post (+ blog) once at startup and runs `minifyCSS` over them — edit the source
  files, never a minified copy, and keep every colour a token (no raw hex/hsl
  outside `:root` except the ink ramp itself).
- Every class a page emits must exist in the inlined CSS. There is a check for
  this pattern in the dev loop: fetch a page, collect `class="..."` tokens, and
  grep the inlined `<style>` for each.
- Images: `PageData.EagerImages` is the page's eager budget (set in
  `newPageData`). `PageData.takeEagerImage()` spends it for avatars, and the
  same counter is handed to the NPF renderer as `Options.EagerBudget` so each
  image spends one slot. Anything past the budget renders `loading="lazy"`. Do
  not hardcode `loading="eager"` in the renderer again.
- `post.js` depends on these hooks — renaming them breaks behaviour silently:
  `.post[data-post-id]`, the first `.blog-name` inside a post (the primary
  author), `.poll-block[data-poll-id]` + `.poll-choices > li[data-answer-id]` +
  `.poll-metadata` (first child is the timestamp), `.populated`,
  `.expired-poll`, and `.control-bar-action` / `.control-bar-dropdown-menu`
  (toggled with the `.open` class).

## Gotchas that have caused real bugs

- **Never percent-decode twice.** `net/http` already decodes `r.URL.Path`, `r.PathValue`
  and the query; decoding again corrupts any tag or query containing a literal `%XX`.
- **`/at/...` must use `App.At`**, a client with redirects disabled, so the upstream `301`
  is visible. The media client follows redirects - using it here returns `200` and the
  resolver always `502`s.
- The API host is hardcoded (`https://www.tumblr.com/api/v2/`). Tests inject a fake via
  the `tumblr.NewAPI(transport http.RoundTripper, ...)` seam - keep that seam and keep the
  fake upstream in `web/integration_test.go` working.
- `web` tests are `package web` and build `App{...}` literally, so unexported `App` fields
  are part of the test surface.
- The build version comes from `-ldflags "-X github.com/dan1165/hyperblur/web.Version="`
  (Dockerfile + CI), falling back to the git revision; don't hardcode it in `main.go`.
- With no `HYPERBLUR_TUMBLR_API_TOKEN`, the bundled token is used and a warning is logged.
- `HYPERBLUR_CACHE=false` disables the page cache entirely (the middleware is skipped).

## Conventions

- CI (`.github/workflows/docker.yml`) runs `go vet` then `go test -race` before building
  the image; a `v*` tag publishes a tagged image, otherwise `latest`.
- Commit messages use short conventional prefixes (`perf:`, `fix:`, `docs:`).
- Env vars and the caching/compression rationale live in `README.md` - keep it in sync
  when behaviour changes.
