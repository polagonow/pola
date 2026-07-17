// @pola/actions/server-reference — client runtime for 'use server' actions.
//
// The bundler rewrites each 'use server' module in the client graph into a set
// of createServerReference(...) exports. Calling one POSTs to /_pola/action and
// returns the server result (or follows a redirect / throws on failure).

export interface ServerActionResponse<T = unknown> {
  success: boolean;
  result?: T;
  error?: string;
  redirect?: string | null;
}

// withCsrf adds the X-CSRF-Token header (read from the <meta name="csrf-token">
// tag the renderer injects) when CSRF protection is active. It's a no-op when
// CSRF is disabled or no token is present.
function withCsrf(headers: Record<string, string>): Record<string, string> {
  if (typeof document !== "undefined") {
    const token = document
      .querySelector('meta[name="csrf-token"]')
      ?.getAttribute("content");
    if (token) headers["X-CSRF-Token"] = token;
  }
  return headers;
}

export function createServerReference(
  actionId: string,
  moduleId: string,
  exportName: string,
): (...args: any[]) => Promise<any> {
  const ref = async (...args: any[]): Promise<any> => {
    const serialized = args.map((arg) =>
      typeof FormData !== "undefined" && arg instanceof FormData
        ? Object.fromEntries(arg.entries())
        : arg,
    );

    const resp = await fetch("/_pola/action", {
      method: "POST",
      headers: withCsrf({ "Content-Type": "application/json" }),
      body: JSON.stringify({ id: moduleId, export_name: exportName, args: serialized }),
    });

    const data: ServerActionResponse = await resp.json().catch(() => ({
      success: false,
      error: `Server action "${exportName}" failed: ${resp.status}`,
    }));

    if (data.redirect) {
      if (typeof window !== "undefined") {
        window.location.assign(data.redirect);
      }
      return { redirect: data.redirect };
    }
    if (!data.success) {
      throw new Error(data.error || `Server action "${exportName}" failed`);
    }
    return data.result;
  };

  // Mark as a server reference so a server component can pass it as a prop and
  // the Flight serializer can bind it on the client.
  (ref as any).$$typeof = Symbol.for("react.server.reference");
  (ref as any).$$id = actionId;
  return ref;
}
