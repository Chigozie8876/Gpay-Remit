// #113 — generic polling hook used by InvoiceViewer for real-time status updates.
import { useEffect, useRef, useState } from "react";

const isDocumentVisible = () =>
  typeof document === "undefined" || document.visibilityState !== "hidden";

/**
 * Calls `callback` immediately then every `intervalMs` milliseconds.
 * Stops when the component unmounts or when `enabled` is false.
 *
 * #351 — by default polling pauses while the tab is hidden and resumes
 * immediately (with a fresh call) when the tab becomes visible again.
 * Pass `{ pauseWhenHidden: false }` to keep polling in background tabs.
 */
export function usePolling(callback, intervalMs = 5000, enabled = true, { pauseWhenHidden = true } = {}) {
  const savedCallback = useRef(callback);
  const [isVisible, setIsVisible] = useState(isDocumentVisible);

  useEffect(() => {
    savedCallback.current = callback;
  }, [callback]);

  useEffect(() => {
    if (!pauseWhenHidden || typeof document === "undefined") return undefined;
    const handleVisibilityChange = () => setIsVisible(isDocumentVisible());
    document.addEventListener("visibilitychange", handleVisibilityChange);
    handleVisibilityChange();
    return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
  }, [pauseWhenHidden]);

  const active = enabled && (!pauseWhenHidden || isVisible);

  useEffect(() => {
    if (!active) return undefined;
    savedCallback.current();
    const id = setInterval(() => savedCallback.current(), intervalMs);
    return () => clearInterval(id);
  }, [intervalMs, active]);
}
