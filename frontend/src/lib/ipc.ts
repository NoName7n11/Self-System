/**
 * IPC bridge — uses Wails runtime when running inside the desktop app,
 * falls back to REST fetch for browser / dev-server mode.
 *
 * After running `wails generate module`, Wails places generated TypeScript
 * bindings in frontend/src/wailsjs/go/. Import from there to get typed
 * wrappers without going through this generic bridge.
 *
 * Detection: Wails injects `window.go` when the WebView is initialised.
 * In browser / Vite dev-server mode, `window.go` is undefined.
 */

/** Returns true when the app is running inside a Wails WebView. */
export function isWailsContext(): boolean {
  return typeof window !== "undefined" && (window as WailsWindow).go !== undefined;
}

type WailsWindow = Window & {
  go?: Record<string, Record<string, Record<string, (...args: unknown[]) => Promise<unknown>>>>;
  runtime?: {
    EventsEmit: (name: string, ...data: unknown[]) => void;
    EventsOn: (name: string, cb: (...data: unknown[]) => void) => () => void;
    EventsOff: (name: string) => void;
  };
};

/**
 * Call a Wails IPC method or fall back to a REST function.
 *
 * @param bindingPath - Dot-separated path matching the generated binding,
 *   e.g. "desktop.App.GetResources" maps to window.go.desktop.App.GetResources.
 * @param args - Arguments forwarded to the IPC method.
 * @param restFallback - Called when not in a Wails context.
 */
export async function ipcCall<T>(
  bindingPath: string,
  args: unknown[],
  restFallback: () => Promise<T>
): Promise<T> {
  if (!isWailsContext()) {
    return restFallback();
  }

  const parts = bindingPath.split(".");
  if (parts.length !== 3) {
    console.warn(`ipcCall: invalid binding path "${bindingPath}", falling back to REST`);
    return restFallback();
  }

  const [pkg, struct, method] = parts;
  const win = window as WailsWindow;
  const fn = win.go?.[pkg]?.[struct]?.[method];

  if (typeof fn !== "function") {
    console.warn(`ipcCall: binding "${bindingPath}" not found, falling back to REST`);
    return restFallback();
  }

  return fn(...args) as Promise<T>;
}

/**
 * Subscribe to a Wails runtime event. No-op in browser mode.
 * Returns an unsubscribe function.
 */
export function onWailsEvent(
  name: string,
  cb: (...data: unknown[]) => void
): () => void {
  const win = window as WailsWindow;
  if (!isWailsContext() || !win.runtime) {
    return () => {};
  }
  return win.runtime.EventsOn(name, cb);
}
