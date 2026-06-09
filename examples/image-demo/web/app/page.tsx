import { type ReactNode } from "react";
import { Image } from "@pola/react/image";
import CdnImage from "../components/CdnImage";

// This page is a React Server Component. <Image> (@pola/react/image) is a client
// component that renders responsive <img>/<srcSet> markup whose URLs point at
// Pola's on-the-fly image optimizer at /_pola/image — enabled by the
// `image_processing` block in Polafile.hcl. The optimizer fetches the remote
// `src`, resizes + re-encodes it per the width/quality query params, caches the
// result, and serves it. Each card below demonstrates one feature.

const PHOTO = "https://picsum.photos/seed/pola-1/1200/800";
const PHOTO2 = "https://picsum.photos/seed/pola-2/1200/800";
const PHOTO3 = "https://picsum.photos/seed/pola-3/1200/800";

// A tiny gray pixel shown (blurred + scaled up) until the real image loads.
const BLUR =
  "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==";

function Demo({ title, caption, children }: { title: string; caption: ReactNode; children: ReactNode }) {
  return (
    <div
      style={{
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-lg)",
        padding: "1rem",
        overflow: "hidden",
      }}
    >
      <h3 style={{ margin: "0 0 0.25rem", fontSize: "1rem" }}>{title}</h3>
      <p style={{ margin: "0 0 0.75rem", color: "var(--color-muted)", fontSize: "0.8125rem" }}>{caption}</p>
      {children}
    </div>
  );
}

export default function HomePage() {
  return (
    <div>
      <div style={{ padding: "2rem 0 1.5rem" }}>
        <h1 style={{ fontSize: "2.25rem", fontWeight: 800, letterSpacing: "-0.02em", margin: 0 }}>
          <code>@pola/react/image</code>
        </h1>
        <p style={{ color: "var(--color-muted)", fontSize: "1.0625rem", marginTop: "0.75rem", maxWidth: 660 }}>
          The <code>{"<Image>"}</code> component renders responsive <code>{"<img>"}</code> markup whose URLs route
          through Pola&apos;s on-the-fly optimizer at <code>/_pola/image</code> (enabled by the{" "}
          <code>image_processing</code> block in <code>Polafile.hcl</code>). Each card shows one feature.
        </p>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: "1rem" }}>
        <Demo
          title="Optimized (fixed size)"
          caption={
            <>
              Resized + re-encoded server-side via <code>/_pola/image?url=…&amp;width=480&amp;quality=75</code>.
            </>
          }
        >
          <Image src={PHOTO} alt="Optimized landscape" width={480} height={320} quality={75} />
        </Demo>

        <Demo title="Quality: 20 vs 85" caption="Same source at two quality levels — note the size/detail trade-off.">
          <div style={{ display: "flex", gap: "0.5rem" }}>
            <Image src={PHOTO} alt="Low quality" width={232} height={155} quality={20} />
            <Image src={PHOTO} alt="High quality" width={232} height={155} quality={85} />
          </div>
        </Demo>

        <Demo
          title="Responsive srcSet"
          caption={
            <>
              No fixed width → a <code>srcSet</code> across device widths; the browser picks one using{" "}
              <code>sizes</code>.
            </>
          }
        >
          <Image
            src={PHOTO2}
            alt="Responsive landscape"
            sizes="(max-width: 600px) 100vw, 480px"
            style={{ width: "100%", height: "auto" }}
          />
        </Demo>

        <Demo
          title="fill"
          caption={
            <>
              Fills a positioned parent with <code>object-fit: cover</code> — handy for fixed-aspect cards.
            </>
          }
        >
          <div style={{ position: "relative", width: "100%", aspectRatio: "3 / 2", borderRadius: 6, overflow: "hidden" }}>
            <Image src={PHOTO3} alt="Filled landscape" fill sizes="(max-width: 600px) 100vw, 480px" />
          </div>
        </Demo>

        <Demo
          title="Blur placeholder"
          caption={
            <>
              Shows a blurred <code>blurDataURL</code> until the full image loads.
            </>
          }
        >
          <Image src={PHOTO2} alt="Blur placeholder" width={480} height={320} placeholder="blur" blurDataURL={BLUR} />
        </Demo>

        <Demo
          title="Custom loader"
          caption={
            <>
              Bypasses <code>/_pola/image</code> — targets <code>images.weserv.nl</code> via the <code>loader</code>{" "}
              prop.
            </>
          }
        >
          <CdnImage src={PHOTO3} alt="Custom loader" width={480} height={320} quality={75} />
        </Demo>

        <Demo
          title="Unoptimized"
          caption={
            <>
              Served as-is (the optimizer is skipped) when you pass <code>unoptimized</code>.
            </>
          }
        >
          <Image src={PHOTO} alt="Unoptimized" width={480} height={320} unoptimized />
        </Demo>
      </div>
    </div>
  );
}
