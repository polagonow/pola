
// Go bridge / runtime globals (injected by runtime/vm.go)
declare const __JSI__: {
    getProducts: () => Array<{ id: number; name: string; price: number; stock: number }>;
    getProduct: (id: string) => { id: number; name: string; price: number; stock: number };
    getUser: (id?: string) => { id: string; name: string; email: string; role: string };
    query: (sql: string, ...args: unknown[]) => unknown[];
};

export declare function fetchJSON(url: string): unknown;

// Server entry (injected by build)
export declare const __CLIENT_MANIFEST__: Record<string, {
    id: string;
    name: string;
    chunks: string[];
    async: boolean;
}>;

export default { ...__JSI__ }
