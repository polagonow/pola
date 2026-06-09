# image-demo

A minimal Pola app showcasing the **`@pola/react/image`** `<Image>` component and
Pola's on-the-fly image optimizer.

## What it shows

`web/app/page.tsx` is a React Server Component that renders `<Image>` (from
`@pola/react/image`) in several modes:

- **Optimized (fixed size)** — resized + re-encoded server-side
- **Quality comparison** — the same source at `quality={20}` vs `quality={85}`
- **Responsive `srcSet`** — omit `width` to emit a width-based `srcSet` + `sizes`
- **`fill`** — fills a positioned parent with `object-fit: cover`
- **Blur placeholder** — `placeholder="blur"` with a `blurDataURL`
- **Custom `loader`** — route through a different CDN (here `images.weserv.nl`)
- **`unoptimized`** — serve the source as-is, skipping the optimizer

Optimized images route through Pola's **`/_pola/image`** endpoint, enabled by the
`image_processing` block in `Polafile.hcl` (the pure-Go `imaging` adapter —
resize, crop, re-encode; no CGO). The component builds URLs like
`/_pola/image?url=<src>&width=<w>&quality=<q>`; the server fetches `src`,
processes it, caches, and serves the result.

## Run

```bash
cd examples/image-demo
pola dev                                  # http://localhost:3000
# or build a single binary:
pola build -o bin/image-demo && ./bin/image-demo
```

The demo fetches placeholder photos from `picsum.photos`, so it needs network
access at runtime — swap the URLs in `web/app/page.tsx` for your own (remote
`http(s)` sources, or use `unoptimized` for local `public/` files).

## Key files

- `Polafile.hcl` — `image_processing { enabled = true, adapter = "imaging" }`
- `web/app/page.tsx` — the `<Image>` showcase

> The `go.mod` uses a relative `replace github.com/polagonow/pola => ../..` so the
> example builds against this checkout of the framework.
