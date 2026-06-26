import { create } from "zustand";

import type { DockTab, GraphView } from "../types";

interface LayoutState {
  leftCollapsed: boolean;
  rightOpen: boolean;
  dockOpen: boolean;
  dockTab: DockTab;
  view: GraphView;
  selectedCat: string | null;

  setLeftCollapsed: (v: boolean) => void;
  toggleLeft: () => void;
  setRightOpen: (v: boolean) => void;
  toggleRight: () => void;
  setDockOpen: (v: boolean) => void;
  openDockTab: (tab: DockTab) => void;
  setDockTab: (tab: DockTab) => void;
  setView: (v: GraphView) => void;
  setSelectedCat: (catId: string | null) => void;
}

export const useLayoutStore = create<LayoutState>((set) => ({
  leftCollapsed: false,
  rightOpen: false,
  dockOpen: false,
  dockTab: "categories",
  view: "graph",
  selectedCat: null,

  setLeftCollapsed: (v) => set({ leftCollapsed: v }),
  toggleLeft: () => set((s) => ({ leftCollapsed: !s.leftCollapsed })),
  setRightOpen: (v) => set({ rightOpen: v }),
  toggleRight: () => set((s) => ({ rightOpen: !s.rightOpen })),
  setDockOpen: (v) => set({ dockOpen: v }),
  openDockTab: (tab) => set({ dockOpen: true, dockTab: tab }),
  setDockTab: (tab) => set({ dockTab: tab }),
  setView: (v) => set({ view: v }),
  setSelectedCat: (catId) => set({ selectedCat: catId }),
}));
