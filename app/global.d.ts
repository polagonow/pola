// Modules without type definitions
declare module "react-server-dom-webpack/server.browser";
declare module 'react-server-dom-esm/client';

// Go bridge / runtime globals (injected by runtime/vm.go)
declare const ctx: {
  getProducts: () => Array<{ id: number; name: string; price: number; stock: number }>;
  getUser: (id?: string) => { id: string; name: string; email: string; role: string };
  query: (sql: string, ...args: unknown[]) => unknown[];
};
declare function fetchJSON(url: string): unknown;

// Server entry (injected by build)
declare const __CLIENT_MANIFEST__: Record<string, {
  id: string;
  name: string;
  chunks: string[];
  async: boolean;
}>;
