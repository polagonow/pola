// @pola/actions/redirect — server-side redirect helper for server actions.
//
// Call redirect(url) inside a 'use server' function to send the client to url.
// It throws a sentinel the Pola action runtime recognizes and converts into a
// { redirect } response (a 303 for form actions).

export function redirect(url: string): never {
  const err = new Error("POLA_REDIRECT") as Error & { __pola_redirect__: string };
  err.__pola_redirect__ = url;
  throw err;
}
