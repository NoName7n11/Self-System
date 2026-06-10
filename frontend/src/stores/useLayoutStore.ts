import { create } from "zustand";

import type { NavSection } from "../types";

interface LayoutState {
  sidebarCollapsed: boolean;
  activeSection: NavSection;
  toggleSidebar: () => void;
  setActiveSection: (section: NavSection) => void;
}

export const useLayoutStore = create<LayoutState>((set) => ({
  sidebarCollapsed: false,
  activeSection: "graph",
  toggleSidebar: () => {
    set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }));
  },
  setActiveSection: (section) => {
    set({ activeSection: section });
  },
}));
