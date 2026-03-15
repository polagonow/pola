/**
 * Custom render wrapper that catches SSRBoundaryError and retries
 * with a React context provider so SSRErrorBoundary can render its fallback.
 *
 * Called from the patched Next.js renderShell function.
 *
 * Supports multiple error boundaries: each iteration catches one new boundary's
 * error, adds it to the errors map, and re-renders. Previously-caught boundaries
 * render their fallback on the next attempt, revealing the next thrower.
 *
 * @param {() => React.ReactElement} makeContent - Creates the React element tree to render
 * @param {(element: React.ReactElement) => Promise<ReadableStream>} doRender - Calls renderToReadableStream
 * @returns {Promise<ReadableStream>} The resulting stream
 */
const React = require("react");

const SSR_ERROR_CONTEXT_KEY = Symbol.for("ssr-error-boundary-context");

const MAX_RETRIES = 10;

/**
 * Returns a singleton React context, shared across module systems via Symbol.for.
 * This ensures that both this file (loaded via Node require from the Next.js patch)
 * and ssr-error-boundary.tsx (bundled by Turbopack) use the same context instance.
 */
function getSSRErrorContext() {
  if (!globalThis[SSR_ERROR_CONTEXT_KEY]) {
    globalThis[SSR_ERROR_CONTEXT_KEY] = React.createContext(null);
  }
  return globalThis[SSR_ERROR_CONTEXT_KEY];
}

async function renderShellWithSSRErrorBoundary(makeContent, doRender) {
  const SSRErrorContext = getSSRErrorContext();

  /** @type {Map<string, Error>} */
  const errors = new Map();

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    try {
      const content = makeContent();
      const element =
        errors.size > 0
          ? React.createElement(
              SSRErrorContext.Provider,
              { value: errors },
              content,
            )
          : content;
      return await doRender(element);
    } catch (error) {
      if (error && error.__isSSRBoundaryError) {
        if (errors.has(error.boundaryId)) {
          // Same boundary threw again — something is wrong, bail out
          throw error;
        }
        errors.set(error.boundaryId, error.originalError);
        continue;
      }
      throw error;
    }
  }

  throw new Error(
    `SSR Error Boundary: exceeded ${MAX_RETRIES} retries. ` +
      `Boundaries that errored: ${[...errors.keys()].join(", ")}`,
  );
}

module.exports = { renderShellWithSSRErrorBoundary, getSSRErrorContext };
