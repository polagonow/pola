// @pola/actions — bridge between Go actions and React server components.
// __DEPENDENCY_INJECTION__ is injected by the Pola runtime per-request.
// Keys are namespaced by struct: "Server.getServerInfo", "Blog.getPosts", etc.
declare const __DEPENDENCY_INJECTION__: Record<string, (...args: any[]) => Promise<any>>

export function createAction(name: string) {
    return new Proxy({} as any, {
        get(_, method: string) {
            const bridge = typeof __DEPENDENCY_INJECTION__ !== 'undefined' ? __DEPENDENCY_INJECTION__ : null
            const key = `${name}.${method}`
            const fn = bridge && bridge[key]
            if (typeof fn === 'function') return (...args: any[]) => fn(...args)
            // No server bridge present. Two cases, with distinct guidance:
            if (bridge === null && typeof window !== 'undefined') {
                // Called from a browser (client component). Go actions run
                // server-side only; wrap them in a "use server" module and import
                // that from the client instead.
                return () =>
                    Promise.reject(
                        new Error(
                            `pola/actions: ${key} was called from a client component, but Go ` +
                                `actions run on the server only. Re-export it from a "use server" ` +
                                `module and import that in your client component, e.g.:\n\n` +
                                `  // lib/actions.ts\n  "use server";\n  import { ${name} } from "@pola/actions";\n` +
                                `  export const ${method} = (...a) => ${name}.${method}(...a);\n\n` +
                                `Then: import { ${method} } from "@/lib/actions" in the client component.`,
                        ),
                    )
            }
            return () => Promise.reject(new Error(`pola/actions: ${key} is not registered`))
        },
    })
}
