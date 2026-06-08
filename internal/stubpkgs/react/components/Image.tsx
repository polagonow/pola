"use client";

import React, { useCallback, useEffect, useRef, useState } from "react";
import { DEFAULT_DEVICE_SIZES } from "./image-constants";

export interface StaticImageData {
  src: string;
  height: number;
  width: number;
  blurDataURL?: string;
}

export interface ImageProps {
  src: string | StaticImageData;
  alt: string;
  width?: number;
  height?: number;
  quality?: number;
  preload?: boolean;
  loading?: "lazy" | "eager";
  placeholder?: "blur" | "empty";
  blurDataURL?: string;
  fill?: boolean;
  sizes?: string;
  style?: React.CSSProperties;
  className?: string;
  onLoad?: (event: React.SyntheticEvent<HTMLImageElement>) => void;
  onError?: (event: React.SyntheticEvent<HTMLImageElement>) => void;
  unoptimized?: boolean;
  loader?: (props: { src: string; width: number; quality: number }) => string;
  overrideSrc?: string;
  decoding?: "async" | "sync" | "auto";
}

// imageEndpoint resolves the image-optimization route. It defaults to the
// reserved /_pola/image path (see core/reserved). When the Polafile sets a
// custom image_processing.path, inject window.__POLA_IMAGE_PATH__ to match, or
// pass a custom `loader` prop.
function imageEndpoint(): string {
  const override = (globalThis as { __POLA_IMAGE_PATH__?: string }).__POLA_IMAGE_PATH__;
  return override || "/_pola/image";
}

// buildImageUrl builds a request to Pola's image endpoint. Pola's "imaging"
// backend produces a single output format, so no `format` parameter is sent
// (unlike rari's avif/webp negotiation); use the `loader` prop to target a
// different optimizer or CDN.
function buildImageUrl(src: string, width: number, quality: number): string {
  const params = new URLSearchParams();
  params.set("url", src);
  params.set("width", String(width));
  params.set("quality", String(quality));
  return `${imageEndpoint()}?${params.toString()}`;
}

/**
 * Image is an optimized <img> wrapper that routes through Pola's image
 * endpoint. Vendored and adapted from rari's <Image> (rari/image): the avif/webp
 * <picture> negotiation is dropped because Pola's image backend emits a single
 * format, and the request shape is mapped to Pola's params (url/width/quality).
 */
export function Image({
  src,
  alt,
  width,
  height,
  quality = 75,
  preload = false,
  loading = "lazy",
  placeholder = "empty",
  blurDataURL,
  fill = false,
  sizes,
  style,
  className,
  onLoad,
  onError,
  unoptimized = false,
  loader,
  overrideSrc,
  decoding,
}: ImageProps) {
  const imgSrc = typeof src === "string" ? src : src.src;
  const imgWidth = width || (typeof src !== "string" ? src.width : undefined);
  const imgHeight = height || (typeof src !== "string" ? src.height : undefined);
  const imgBlurDataURL = blurDataURL || (typeof src !== "string" ? src.blurDataURL : undefined);
  const finalSrc = overrideSrc || imgSrc;
  const shouldPreload = preload;
  const imgDecoding = decoding || (preload ? "sync" : "async");

  const [blurComplete, setBlurComplete] = useState(false);
  const [showAltText, setShowAltText] = useState(false);
  const imgRef = useRef<HTMLImageElement>(null);
  const onLoadRef = useRef(onLoad);

  useEffect(() => {
    onLoadRef.current = onLoad;
  }, [onLoad]);

  const resolveSrc = useCallback(
    (w: number) =>
      loader ? loader({ src: finalSrc, width: w, quality }) : buildImageUrl(finalSrc, w, quality),
    [loader, finalSrc, quality],
  );

  const handleLoad = useCallback(
    (event: React.SyntheticEvent<HTMLImageElement>) => {
      const img = event.currentTarget;
      if (img.src && img.complete) {
        if (placeholder === "blur") setBlurComplete(true);
        if (onLoadRef.current) onLoadRef.current(event);
      }
    },
    [placeholder],
  );

  const handleError = useCallback(
    (event: React.SyntheticEvent<HTMLImageElement>) => {
      setShowAltText(true);
      if (placeholder === "blur") setBlurComplete(true);
      if (onError) onError(event);
    },
    [placeholder, onError],
  );

  // Preload the primary candidate when preload is requested.
  useEffect(() => {
    if (!shouldPreload) return;
    const link = document.createElement("link");
    link.rel = "preload";
    link.as = "image";
    link.href = unoptimized ? finalSrc : resolveSrc(imgWidth || 1920);
    if (sizes) link.setAttribute("imagesizes", sizes);
    document.head.appendChild(link);
    return () => {
      if (link.parentNode === document.head) document.head.removeChild(link);
    };
  }, [shouldPreload, finalSrc, imgWidth, sizes, unoptimized, resolveSrc]);

  const imgStyle: React.CSSProperties = {
    ...style,
    ...(fill && {
      position: "absolute",
      inset: 0,
      width: "100%",
      height: "100%",
      objectFit: "cover",
    }),
    ...(placeholder === "blur" &&
      imgBlurDataURL &&
      !blurComplete && {
        backgroundImage: `url(${imgBlurDataURL})`,
        backgroundSize: "cover",
        backgroundPosition: "center",
        filter: "blur(20px)",
        transition: "filter 0.3s ease-out",
      }),
    ...(placeholder === "blur" &&
      blurComplete && {
        filter: "none",
        transition: "filter 0.3s ease-out",
      }),
  };

  if (unoptimized) {
    const finalImgSrc = loader ? loader({ src: finalSrc, width: imgWidth || 1920, quality }) : finalSrc;
    return (
      <img
        ref={imgRef}
        src={finalImgSrc}
        alt={showAltText ? alt : ""}
        width={fill ? undefined : imgWidth}
        height={fill ? undefined : imgHeight}
        loading={shouldPreload ? "eager" : loading}
        fetchPriority={shouldPreload ? "high" : "auto"}
        decoding={imgDecoding}
        onLoad={handleLoad}
        onError={handleError}
        style={imgStyle}
        className={className}
      />
    );
  }

  const defaultWidth = imgWidth || 1920;
  const sizesArray = imgWidth ? [imgWidth] : DEFAULT_DEVICE_SIZES;
  const shouldUseSrcSet = sizesArray.length > 1 || sizesArray[0] !== defaultWidth;
  const srcSet = shouldUseSrcSet ? sizesArray.map((w) => `${resolveSrc(w)} ${w}w`).join(", ") : undefined;

  return (
    <img
      ref={imgRef}
      src={resolveSrc(defaultWidth)}
      srcSet={srcSet}
      sizes={shouldUseSrcSet ? sizes : undefined}
      alt={showAltText ? alt : ""}
      width={fill ? undefined : imgWidth}
      height={fill ? undefined : imgHeight}
      loading={shouldPreload ? "eager" : loading}
      fetchPriority={shouldPreload ? "high" : "auto"}
      decoding={imgDecoding}
      onLoad={handleLoad}
      onError={handleError}
      style={imgStyle}
      className={className}
    />
  );
}

export default Image;
