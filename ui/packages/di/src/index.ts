/// <reference path="./generated.d.ts" />

// @pola/di — bridge between Go services and React server components
// __DEPENDENCY_INJECTION__ is injected by the Pola runtime per-request.
// A Proxy is used so di.foo() always reads from the current global at call
// time rather than capturing a stale reference at module-evaluation time.
declare const __DEPENDENCY_INJECTION__: Record<string, (...args: any[]) => Promise<any>>

const di = new Proxy({} as Record<string, (...args: any[]) => Promise<any>>, {
    get(_, name: string) {
        const bridge = typeof __DEPENDENCY_INJECTION__ !== 'undefined' ? __DEPENDENCY_INJECTION__ : null
        const fn = bridge && (bridge as any)[name as string]
        if (typeof fn === 'function') return (...args: any[]) => fn(...args)
        return () => Promise.reject(new Error(`pola/di: ${String(name)} not registered`))
    },
})

export default di
