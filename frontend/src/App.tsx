import { useEffect } from "react";

import GraphCanvas from "./components/graph/GraphCanvas";
import Dock from "./components/chat/ChatDock";
import Inspector from "./components/resource/ResourceForm";
import Sidebar from "./components/layout/Sidebar";
import Topbar from "./components/layout/Topbar";
import { useFileDrop } from "./hooks/useFileDrop";
import { useFilteredResources } from "./hooks/useFilteredResources";
import { useResourceStore } from "./stores/useResourceStore";
import { useSyncStore } from "./stores/useSyncStore";
import { useTaskStore } from "./stores/useTaskStore";

export default function App() {
  const loadResources = useResourceStore((s) => s.loadResources);
  const loadTasks = useTaskStore((s) => s.loadAll);
  const allResources = useResourceStore((s) => s.resources);
  const startSync = useSyncStore((s) => s.start);
  const stopSync = useSyncStore((s) => s.stop);
  const filteredResources = useFilteredResources();

  useFileDrop();

  useEffect(() => { void loadResources(); }, [loadResources]);
  useEffect(() => { void loadTasks(); }, [loadTasks]);
  useEffect(() => {
    startSync();
    return () => stopSync();
  }, [startSync, stopSync]);

  return (
    <div className="app-shell">
      <Sidebar />

      <div className="center-col">
        <Topbar resourceCount={allResources.length} />
        <GraphCanvas resources={filteredResources} />
        <Dock />
      </div>

      <Inspector />
    </div>
  );
}
