// #105 — catches render errors in the component tree and shows a friendly
// fallback instead of a blank white screen.
// #354 — adds Try again / Reload page actions, reports the error, and shows
// the raw error message only in development.
import React from "react";
import { reportError } from "../utils/errorReporting";

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, message: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, message: error?.message || "Unknown error" };
  }

  componentDidCatch(error, info) {
    reportError(error, info);
  }

  handleReset = () => {
    this.setState({ hasError: false, message: null });
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      const isDevelopment = process.env.NODE_ENV === "development";
      return (
        <div role="alert" className="error-boundary" aria-live="assertive">
          <h2>Something went wrong</h2>
          <p>Try again, or reload the page if the problem continues.</p>
          {isDevelopment && this.state.message && (
            <p className="error-boundary-detail">{this.state.message}</p>
          )}
          <div className="error-boundary-actions" style={{ display: "flex", gap: "10px" }}>
            <button type="button" onClick={this.handleReset} aria-label="Try again">
              Try again
            </button>
            <button type="button" onClick={this.handleReload} aria-label="Reload page">
              Reload page
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

export default ErrorBoundary;
