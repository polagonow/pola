// Server render for the control: React's streaming SSR primitive,
// renderToPipeableStream — no framework, no RSC.

import React, { Suspense } from "react";
import { renderToPipeableStream } from "react-dom/server";
import { Document, StaticBody, AsyncBody, AsyncFallback, IslandBody } from "./App.jsx";

const delay = (ms, value) => new Promise((resolve) => setTimeout(() => resolve(value), ms));

// Renders one scenario into `res` (a Node http.ServerResponse).
export function renderScenario(id, res) {
  let element;
  let bootstrapModules = [];

  if (id === "1") {
    element = (
      <Document title="Scenario 1 — Static">
        <StaticBody />
      </Document>
    );
  } else if (id === "2") {
    // Fresh 50ms promise per request → real per-request async latency, streamed
    // out-of-band once it resolves (shell flushes first with the fallback).
    const dataPromise = delay(50, "Loaded after 50ms");
    element = (
      <Document title="Scenario 2 — Async (50ms, streamed)">
        <Suspense fallback={<AsyncFallback />}>
          <AsyncBody dataPromise={dataPromise} />
        </Suspense>
      </Document>
    );
  } else if (id === "3") {
    bootstrapModules = ["/client/s3.js"];
    element = (
      <Document title="Scenario 3 — Interactive Island">
        <IslandBody />
      </Document>
    );
  } else {
    res.statusCode = 404;
    res.end("unknown scenario");
    return;
  }

  const stream = renderToPipeableStream(element, {
    bootstrapModules,
    onShellReady() {
      res.statusCode = 200;
      res.setHeader("content-type", "text/html; charset=utf-8");
      stream.pipe(res);
    },
    onShellError() {
      res.statusCode = 500;
      res.setHeader("content-type", "text/html; charset=utf-8");
      res.end("<!doctype html><p>shell error</p>");
    },
    onError(err) {
      process.stderr.write("render error: " + String(err) + "\n");
    },
  });
}
