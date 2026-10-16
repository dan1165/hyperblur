# AGENTS.md

hyperblur — Tumblr frontend. One Go binary, assets embedded, no web framework.

## Layout

- `cmd/hyperblur/main.go` — entrypoint: flags, `http.Server`, graceful shutdown.
- `web/` — HTTP layer: routing + media (`server.go`), handlers, views, `cache.go`, `version.go`.
- `web/assets/` — CSS/JS/PWA files, embedded with `//go:embed assets`. Put assets here, not at repo root.
- `tumblr/` — Tumblr API client, models, JSON parsing.
- `npf/` — Neue Post Format to HTML renderer.

One module (`github.com/dan1165/hyperblur`), Go 1.27. `go.mod`, Dockerfile and CI must stay in step.

## Commands

```bash
go vet ./...                                  # CI runs this
go test ./...                                 # unit + integration
go test -race ./...                           # CI runs this (needs cgo)
go test ./web/ -run TestPageCache                # one package / one test
go build -o bin/hyperblur ./cmd/hyperblur     # then ./bin/hyperblur  (listens :8000)
```

Build into `bin/` on Windows: `.gitignore` covers `/bin/` and `/hyperblur`, but not `hyperblur.exe`.

`hyperblur -healthcheck` probes local `/healthz`, sets exit code. That is the
compose healthcheck (image is `scratch`: no shell, no curl).

## Routing & cache (easy to break)

- ServeMux patterns cover `/assets/`, `/tblr/...`, `/at/{path...}`, `/api/v1/poll/...`,
  `/healthz`, `/{$}`. Everything else (explore/search/tagged/blog) is hand-split in
  `handleFallback` (`GET /`) — blog paths collide with the wildcard patterns, so do not
  move them into the mux.
- `cacheablePath` (`web/cache.go`) decides what the page cache may store. It excludes
  `/assets/`, `/tblr/`, `/api/`, `/at/`, `/robots.txt`, `/manifest.webmanifest`,
  `/sw.js`, `/healthz`. Any other `200` HTML page is cached automatically.
- A cached page stores **raw HTML + one precomputed gzip**. Client encoding is
  negotiated at serve time; cache key is just `path?query`. `writePage` deliberately
  writes uncompressed for cacheable paths so the cache can do the encoding — do not
  re-add gzip there.
- Server intentionally has **no `WriteTimeout`**: media streams, a deadline would cut
  long downloads.

## UI layer (austere monochrome)

- Stylesheets: `web/assets/css/{base,post,blog}.css`. `base.css` holds token block
  (`--ink-*`, spacing, motion, type scale), shell, nav, controls, paging. `post.css`
  holds posts plus whole NPF block set. `blog.css` appended on blog pages only.
  `inlineCSS` (`web/server.go`) concatenates base + post (+ blog) once at startup and
  runs `minifyCSS` over them. Edit source files, never a minified copy. Keep every
  colour a token: no raw hex/hsl outside `:root`, except ink ramp itself.
- Every class a page emits must exist in the inlined CSS. Dev-loop check: fetch a
  page, collect `class="..."` tokens, grep inlined `<style>` for each.
- Images: `PageData.EagerImages` is page's eager budget (set in `newPageData`).
  `PageData.takeEagerImage()` spends it for avatars. Same counter handed to NPF
  renderer as `Options.EagerBudget`, so each image spends one slot. Anything past
  budget renders `loading="lazy"`. Do not hardcode `loading="eager"` in renderer again.
- `post.js` depends on these hooks — renaming breaks behaviour silently:
  `.post[data-post-id]`, first `.blog-name` inside a post (primary author),
  `.poll-block[data-poll-id]` + `.poll-choices > li[data-answer-id]` +
  `.poll-metadata` (first child is timestamp), `.populated`, `.expired-poll`, and
  `.control-bar-action` / `.control-bar-dropdown-menu` (toggled with `.open` class).

## Gotchas that have caused real bugs

- **Never percent-decode twice.** `net/http` already decodes `r.URL.Path`, `r.PathValue`
  and the query. Decoding again corrupts any tag or query containing literal `%XX`.
- **`/at/...` must use `App.At`**, a client with redirects disabled, so upstream `301`
  is visible. Media client follows redirects: using it here returns `200` and resolver
  always `502`s.
- API host is hardcoded (`https://www.tumblr.com/api/v2/`). Tests inject a fake via the
  `tumblr.NewAPI(transport http.RoundTripper, ...)` seam. Keep that seam, keep fake
  upstream in `web/integration_test.go` working.
- `web` tests are `package web` and build `App{...}` literally, so unexported `App`
  fields are part of test surface.
- Build version comes from `-ldflags "-X github.com/dan1165/hyperblur/web.Version="`
  (Dockerfile + CI), falling back to git revision. Do not hardcode it in `main.go`.
- With no `HYPERBLUR_TUMBLR_API_TOKEN`, bundled token is used and a warning is logged.
- `HYPERBLUR_CACHE=false` disables page cache entirely (middleware is skipped).

## Conventions

- CI (`.github/workflows/docker.yml`) runs `go vet`, then `go test -race`, before
  building image. A `v*` tag publishes tagged image, otherwise `latest`.
- Commit messages use short conventional prefixes (`perf:`, `fix:`, `docs:`).
- Env vars and caching/compression rationale live in `README.md`. Keep it in sync when
  behaviour changes.
