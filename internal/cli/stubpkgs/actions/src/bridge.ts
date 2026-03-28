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
            return () => Promise.reject(new Error(`pola/actions: ${key} not registered`))
        },
    })
}
