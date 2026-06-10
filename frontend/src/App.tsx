import { useEffect } from "react";

import ChatDock from "./components/chat/ChatDock";
import GraphCanvas from "./components/graph/GraphCanvas";
import GraphControls from "./components/graph/GraphControls";
import Sidebar from "./components/layout/Sidebar";
import Topbar from "./components/layout/Topbar";
import SettingsPanel from "./components/settings/SettingsPanel";
import ResourceForm from "./components/resource/ResourceForm";
import ResourceList from "./components/resource/ResourceList";
import TaskBoard from "./components/tasks/TaskBoard";
import { useFileDrop } from "./hooks/useFileDrop";
import { useFilteredResources } from "./hooks/useFilteredResources";
import { useLayoutStore } from "./stores/useLayoutStore";
import { useResourceStore } from "./stores/useResourceStore";
import { useSyncStore } from "./stores/useSyncStore";
import { useTaskStore } from "./stores/useTaskStore";

export default function App() {
  const loadResources = useResourceStore((state) => state.loadResources);
  const loadTasks = useTaskStore((state) => state.loadAll);
  const allResources = useResourceStore((state) => state.resources);
  const startSync = useSyncStore((state) => state.start);
  const stopSync = useSyncStore((state) => state.stop);
  const sidebarCollapsed = useLayoutStore((state) => state.sidebarCollapsed);
  const activeSection = useLayoutStore((state) => state.activeSection);
  const filteredResources = useFilteredResources();

  useFileDrop();

  useEffect(() => {
    void loadResources();
  }, [loadResources]);

  useEffect(() => {
    void loadTasks();
  }, [loadTasks]);

  useEffect(() => {
    startSync();

    return () => {
      stopSync();
    };
  }, [startSync, stopSync]);

  return (
    <div className={`app-shell ${sidebarCollapsed ? "sidebar-collapsed" : ""}`}>
      <Sidebar />

      <div className="workspace">
        <Topbar totalResources={allResources.length} visibleResources={filteredResources.length} />

        {activeSection === "graph" ? (
          <main className="app-content">
            <section className="column-main">
              <GraphControls resources={allResources} />
              <GraphCanvas resources={filteredResources} />
              <ResourceList resources={filteredResources} />
            </section>

            <section className="column-side">
              <ResourceForm />
              <ChatDock />
            </section>
          </main>
        ) : null}

        {activeSection === "search" ? (
          <main className="app-content">
            <section className="column-main">
              <GraphControls resources={allResources} />
              <ResourceList resources={filteredResources} />
            </section>

            <section className="column-side">
              <ResourceForm />
            </section>
          </main>
        ) : null}

        {activeSection === "chat" ? (
          <main className="app-content">
            <section className="column-main">
              <ChatDock />
              <ResourceList resources={filteredResources} />
            </section>

            <section className="column-side">
              <ResourceForm />
            </section>
          </main>
        ) : null}

        {activeSection === "tasks" ? (
          <main className="app-content app-content-wide">
            <section className="column-main">
              <TaskBoard />
            </section>

            <section className="column-side">
              <ChatDock />
            </section>
          </main>
        ) : null}

        {activeSection === "settings" ? (
          <main className="app-content app-content-wide">
            <section className="column-main">
              <SettingsPanel />
            </section>

            <section className="column-side">
              <ResourceForm />
            </section>
          </main>
        ) : null}
      </div>
    </div>
  );
}
