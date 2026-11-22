// #354 — forwards errors caught by ErrorBoundary to an error reporting endpoint.
// The endpoint is optional: set REACT_APP_ERROR_REPORT_URL to enable it.
// Reporting must never throw or block the UI.
const REPORT_URL = process.env.REACT_APP_ERROR_REPORT_URL;

export function reportError(error, info = {}) {
  console.error("[ErrorBoundary]", error, info.componentStack);

  if (!REPORT_URL || typeof fetch !== "function") return;

  const payload = {
    message: error?.message || "Unknown error",
    stack: error?.stack || null,
    componentStack: info.componentStack || null,
    url: typeof window !== "undefined" ? window.location.href : null,
    timestamp: new Date().toISOString(),
  };

  try {
    fetch(REPORT_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
      keepalive: true,
    }).catch(() => {});
  } catch {
    // ignore reporting failures
  }
}
