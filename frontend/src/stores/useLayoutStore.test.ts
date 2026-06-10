import { beforeEach, describe, expect, it } from "vitest";

import { useLayoutStore } from "./useLayoutStore";

describe("useLayoutStore", () => {
  beforeEach(() => {
    useLayoutStore.setState({
      sidebarCollapsed: false,
      activeSection: "graph",
    });
  });

  it("starts with the default layout state", () => {
    const state = useLayoutStore.getState();
    expect(state.sidebarCollapsed).toBe(false);
    expect(state.activeSection).toBe("graph");
  });

  it("toggles the sidebar collapsed flag", () => {
    useLayoutStore.getState().toggleSidebar();
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(true);

    useLayoutStore.getState().toggleSidebar();
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(false);
  });

  it("sets the active section", () => {
    useLayoutStore.getState().setActiveSection("tasks");

    const state = useLayoutStore.getState();
    expect(state.activeSection).toBe("tasks");
  });

  it("does not change the active section when toggling", () => {
    useLayoutStore.getState().setActiveSection("chat");
    useLayoutStore.getState().toggleSidebar();

    const state = useLayoutStore.getState();
    expect(state.activeSection).toBe("chat");
  });
});
