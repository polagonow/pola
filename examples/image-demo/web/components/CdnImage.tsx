"use client";

import { Image } from "@pola/react/image";

// A custom `loader` must live in a client component: functions cannot be passed
// from a Server Component to a client component across the RSC boundary. This
// loader routes <Image> through the public images.weserv.nl proxy instead of
// Pola's /_pola/image endpoint.
function weservLoader({ src, width, quality }: { src: string; width: number; quality: number }) {
  return `https://images.weserv.nl/?url=${encodeURIComponent(src)}&w=${width}&q=${quality}`;
}

export default function CdnImage(props: {
  src: string;
  alt: string;
  width: number;
  height: number;
  quality?: number;
}) {
  return <Image {...props} loader={weservLoader} />;
}
