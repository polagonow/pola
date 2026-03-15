// Framework-injected globals — available in all server components.

declare global {
  // Per-request context injected by the Go VM before each render.
  const __request__: {
    url: string;
    path: string;
    query: string;
    method: string;
    headers: Record<string, string>;
  };
}

// Client manifest injected as a JS define by the bundler.
export declare const __CLIENT_MANIFEST__: Record<string, {
  id: string;
  name: string;
  chunks: string[];
  async: boolean;
}>;
