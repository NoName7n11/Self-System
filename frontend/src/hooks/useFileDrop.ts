import { useEffect } from "react";

import { ipcCall, onWailsEvent } from "../lib/ipc";
import { listResources } from "../api/client";
import { useResourceStore } from "../stores/useResourceStore";

/** Extracts a display title from a dropped file's absolute path. */
function basename(path: string): string {
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] || path;
}

/**
 * Subscribes to the native "files:dropped" event emitted by the Wails backend
 * (registered via runtime.OnFileDrop in Startup) and creates one resource per
 * dropped file by calling the CreateResource IPC binding. No-op in browser
 * mode, where Wails file-drop events never fire.
 */
export function useFileDrop(): void {
  const loadResources = useResourceStore((state) => state.loadResources);

  useEffect(() => {
    const unsubscribe = onWailsEvent("files:dropped", (...data: unknown[]) => {
      const paths = (data[0] as string[]) ?? [];
      if (!Array.isArray(paths) || paths.length === 0) {
        return;
      }

      void Promise.all(
        paths.map((path) =>
          ipcCall<unknown>(
            "desktop.App.CreateResource",
            [path, basename(path), "", ""],
            // File drop only happens inside Wails; the REST fallback is never
            // reached, but listResources keeps the signature total.
            () => listResources()
          )
        )
      ).then(() => loadResources({ silent: true }));
    });

    return unsubscribe;
  }, [loadResources]);
}
