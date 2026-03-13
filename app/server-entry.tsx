import React from "react";
import { renderToReadableStream } from "react-server-dom-webpack/server.browser";
import { IndexPage }    from "./pages/index";
import { ProductsPage } from "./pages/products";
import { UserPage }     from "./pages/user";

const pages: Record<string, React.ComponentType<Record<string, unknown>>> = {
  IndexPage,
  ProductsPage,
  UserPage,
};

// renderToReadableStream from react-server-dom-webpack/server.browser returns
// a ReadableStream directly (NOT a Promise) — stream is ready synchronously.
(globalThis as any).__render__ = function (
  exportName: string,
  propsJSON: string,
): ReadableStream {
  const Page = pages[exportName];
  if (!Page) {
    throw new Error(
      `__render__: unknown page "${exportName}". Known: ${Object.keys(pages).join(", ")}`,
    );
  }
  const props = JSON.parse(propsJSON || "{}");
  return renderToReadableStream(
    React.createElement(Page, props),
    __CLIENT_MANIFEST__,
  );
};
