import React, { createContext, useContext } from "react";

// --- Shared context (singleton via Symbol.for, see ssr-render.js) ---

type SSRErrors = Map<string, Error> | null;

const SSR_ERROR_CONTEXT_KEY = Symbol.for("ssr-error-boundary-context");
const _g = globalThis as Record<symbol, unknown>;

function getSSRErrorContext(): React.Context<SSRErrors> {
  if (!_g[SSR_ERROR_CONTEXT_KEY]) {
    _g[SSR_ERROR_CONTEXT_KEY] = createContext<SSRErrors>(null);
  }
  return _g[SSR_ERROR_CONTEXT_KEY] as React.Context<SSRErrors>;
}

// --- SSRBoundaryError ---

export class SSRBoundaryError extends Error {
  __isSSRBoundaryError = true;
  boundaryId: string;
  originalError: Error;

  constructor(message: string, boundaryId: string) {
    super(message);
    this.name = "SSRBoundaryError";
    this.boundaryId = boundaryId;
    this.originalError = new Error(message);
  }
}

// --- SSRBoundaryIdContext (passes boundary ID to descendant hooks) ---

const SSRBoundaryIdContext = createContext<string | null>(null);

// --- useThrowToParentErrorBoundary ---

export function useThrowToParentErrorBoundary(): (message: string) => never {
  const boundaryId = useContext(SSRBoundaryIdContext);
  return (message: string) => {
    if (!boundaryId) {
      throw new Error(
        `useThrowToParentErrorBoundary: No SSRErrorBoundary ancestor found. ` +
          `Wrap this component in an <SSRErrorBoundary>.`,
      );
    }
    throw new SSRBoundaryError(message, boundaryId);
  };
}

// --- SSRErrorBoundary ---

interface SSRErrorBoundaryProps {
  id: string;
  fallback: React.ReactNode;
  children: React.ReactNode;
}

interface SSRErrorBoundaryState {
  hasError: boolean;
}

export class SSRErrorBoundary extends React.Component<
  SSRErrorBoundaryProps,
  SSRErrorBoundaryState
> {
  constructor(props: SSRErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): SSRErrorBoundaryState {
    return { hasError: true };
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback;
    }

    const SSRErrorContext = getSSRErrorContext();

    return (
      <SSRErrorContext.Consumer>
        {(ssrErrors) => {
          if (ssrErrors && ssrErrors.has(this.props.id)) {
            return this.props.fallback;
          }
          return (
            <SSRBoundaryIdContext.Provider value={this.props.id}>
              {this.props.children}
            </SSRBoundaryIdContext.Provider>
          );
        }}
      </SSRErrorContext.Consumer>
    );
  }
}
