import { useEffect } from "react";

import { ipcCall, onWailsFileDrop } from "../lib/ipc";
import { listResources } from "../api/client";
import { useResourceStore } from "../stores/useResourceStore";

/** Extracts a display title from a dropped file's absolute path. */
function basename(path: string): string {
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] || path;
}

/**
 * Registers the native Wails file-drop handler and creates one resource per
 * dropped file via the CreateResource IPC binding. No-op in browser mode, where
 * Wails' runtime (and thus OnFileDrop) is absent.
 */
export function useFileDrop(): void {
  const loadResources = useResourceStore((state) => state.loadResources);

  useEffect(() => {
    const unsubscribe = onWailsFileDrop((_x, _y, paths) => {
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
