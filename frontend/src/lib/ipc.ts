/**
 * IPC bridge — uses Wails runtime when running inside the desktop app,
 * falls back to REST fetch for browser / dev-server mode.
 *
 * After running `wails generate module`, Wails places generated TypeScript
 * bindings in frontend/wailsjs/go/ (canonical Wails location, regenerated on
 * every `wails build`). Import from there (e.g. `../../wailsjs/go/desktop/App`)
 * to get typed wrappers without going through this generic bridge.
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
    OnFileDrop: (
      cb: (x: number, y: number, paths: string[]) => void,
      useDropTarget: boolean
    ) => void;
    OnFileDropOff: () => void;
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

/**
 * Register a native file-drop handler. No-op in browser mode. Returns an
 * unsubscribe function.
 *
 * This calls Wails' JS-side `window.runtime.OnFileDrop`, which attaches the DOM
 * dragover/drop listeners that `preventDefault()` the drop and post the dropped
 * file objects to Go. Without this call the WebView2 default handler wins and
 * opens the file (e.g. a PDF in an Edge viewer) instead of delivering paths.
 * The Go-side `runtime.OnFileDrop` only subscribes to the resulting event — it
 * does NOT attach these listeners — so the frontend must register here.
 *
 * `useDropTarget=false` makes the entire window a drop zone, so no element needs
 * the `--wails-drop-target: drop` CSS marker.
 */
export function onWailsFileDrop(
  cb: (x: number, y: number, paths: string[]) => void
): () => void {
  const win = window as WailsWindow;
  if (!isWailsContext() || !win.runtime?.OnFileDrop) {
    return () => {};
  }
  win.runtime.OnFileDrop(cb, false);
  return () => win.runtime?.OnFileDropOff?.();
}
