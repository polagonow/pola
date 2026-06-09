/**
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 *
 * Vendored from: https://github.com/facebook/react
 * Original file: packages/react-client/src/ReactFlightClientConfig.js (shim)
 * Reference impl: packages/react-server-dom-webpack/src/client/ReactFlightClientConfigBundlerWebpack.js
 * Modifications: Adapted for Pola — client references resolve via native ESM
 *   dynamic import() (the chunk URLs come from the page's <script type="importmap">)
 *   rather than "~clientComponents" global registry or webpack's chunk loader.
 *
 * Pola's server (react-server-dom-webpack/server.browser) emits client-reference
 * metadata as the array [id, chunks, name], e.g. ["components/ThemeToggle",
 * ["default"], "default"]. `id` is an importmap key that maps to the hashed
 * browser chunk URL, so `import(id)` loads the module and `name` selects the export.
 */

export interface ClientReference<T> {
  specifier: string;
  name: string;
}

// Pola emits [id, chunks, name]; an { id, chunks, name } object form is also
// accepted defensively. Typed loosely because the value comes straight off the
// Flight wire.
export type ClientReferenceMetadata = any;

export type ServerConsumerModuleMap = Record<string, any>;

export interface StringDecoder {
  decode: (chunk: Uint8Array, options?: { stream?: boolean }) => string;
}

export function resolveClientReference<T>(metadata: ClientReferenceMetadata): ClientReference<T> {
  if (Array.isArray(metadata)) {
    // [id, chunks, name] — prefer the explicit name, fall back to chunks[0].
    const name =
      typeof metadata[2] === "string" && metadata[2]
        ? metadata[2]
        : Array.isArray(metadata[1])
          ? String(metadata[1][0] ?? "default")
          : "default";
    return { specifier: String(metadata[0]), name };
  }
  return { specifier: String(metadata.id), name: metadata.name || "default" };
}

interface ModuleEntry {
  promise?: Promise<any>;
  module?: any;
}

// Cache of in-flight / resolved dynamic imports, keyed by specifier.
const moduleCache = new Map<string, ModuleEntry>();

export function preloadModule<T>(moduleReference: ClientReference<T>): null | Promise<any> {
  const specifier = moduleReference.specifier;
  let entry = moduleCache.get(specifier);
  if (!entry) {
    entry = {};
    moduleCache.set(specifier, entry);
    const current = entry;
    current.promise = import(/* @vite-ignore */ specifier).then(
      (mod: any) => {
        current.module = mod;
        return mod;
      },
      (err: any) => {
        moduleCache.delete(specifier);
        throw err;
      },
    );
  }
  if (entry.module) {
    return null;
  }
  return entry.promise ?? null;
}

export function requireModule<T>(moduleReference: ClientReference<T>): T {
  let entry = moduleCache.get(moduleReference.specifier);
  if (!entry || !entry.module) {
    // Not resolved yet — kick off the load and suspend on its promise.
    const promise = preloadModule(moduleReference);
    if (promise) {
      throw promise;
    }
    entry = moduleCache.get(moduleReference.specifier);
    if (!entry || !entry.module) {
      throw new Error(`[pola] client module not loaded: ${moduleReference.specifier}`);
    }
  }

  const mod: any = entry.module;
  const name = moduleReference.name;
  if (name === "default") {
    return (mod.default !== undefined ? mod.default : mod) as T;
  }
  if (mod[name] !== undefined) {
    return mod[name] as T;
  }
  if (mod.default && mod.default[name] !== undefined) {
    return mod.default[name] as T;
  }
  return mod as T;
}

export function readPartialStringChunk(decoder: StringDecoder, buffer: Uint8Array): string {
  return decoder.decode(buffer, { stream: true });
}

export function readFinalStringChunk(decoder: StringDecoder, buffer: Uint8Array): string {
  return decoder.decode(buffer);
}

export function createStringDecoder(): StringDecoder {
  return new TextDecoder();
}
