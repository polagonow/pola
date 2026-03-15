"use client";
import React from "react";

type Props = {
  fallback: React.ComponentType<{ error: Error; reset: () => void }>;
  children: React.ReactNode;
};
type State = { error: Error | null };

export default class ErrorBoundary extends React.Component<Props, State> {
  state: State = { error: null };
  static getDerivedStateFromError(error: Error): State { return { error }; }
  render() {
    if (this.state.error) {
      const reset = () => this.setState({ error: null });
      return React.createElement(this.props.fallback, {
        error: this.state.error,
        reset,
      });
    }
    return this.props.children;
  }
}
