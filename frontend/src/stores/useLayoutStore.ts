import { create } from "zustand";

import type { DockTab, GraphView, LeftView } from "../types";

interface LayoutState {
  leftCollapsed: boolean;
  rightOpen: boolean;
  dockOpen: boolean;
  dockTab: DockTab;
  dockConvId: string;
  view: GraphView;
  leftView: LeftView;
  selectedCat: string | null;
  recentOpen: boolean;
  catsOpen: boolean;
  libFilter: string;
  archiveFilter: string;
  notifOpen: boolean;
  notifSeen: boolean;
  notifMuted: boolean;
  mapExpanded: Record<string, boolean>;
  mapAllExpanded: boolean;
  mapZoom: number;

  toggleLeft: () => void;
  setLeftCollapsed: (v: boolean) => void;
  toggleRight: () => void;
  setRightOpen: (v: boolean) => void;
  toggleDock: () => void;
  setDockOpen: (v: boolean) => void;

  setLeftView: (v: LeftView) => void;
  openDockTab: (tab: DockTab) => void;
  setDockTab: (tab: DockTab) => void;
  openConvInDock: (convId: string) => void;
  setDockConvId: (convId: string) => void;
  setView: (v: GraphView) => void;

  setSelectedCat: (catId: string | null) => void;
  toggleRecent: () => void;
  toggleCats: () => void;
  setLibFilter: (f: string) => void;
  setArchiveFilter: (f: string) => void;

  toggleNotif: () => void;
  closeNotif: () => void;
  toggleMute: () => void;

  toggleMapNode: (id: string) => void;
  toggleMapAll: () => void;
  mapZoomIn: () => void;
  mapZoomOut: () => void;
  mapZoomReset: () => void;
}

export const useLayoutStore = create<LayoutState>((set) => ({
  leftCollapsed: false,
  rightOpen: true,
  dockOpen: true,
  dockTab: "categories",
  dockConvId: "cv1",
  view: "graph",
  leftView: "home",
  selectedCat: null,
  recentOpen: true,
  catsOpen: true,
  libFilter: "all",
  archiveFilter: "all",
  notifOpen: false,
  notifSeen: false,
  notifMuted: false,
  mapExpanded: {},
  mapAllExpanded: false,
  mapZoom: 1,

  toggleLeft: () => set((s) => ({ leftCollapsed: !s.leftCollapsed })),
  setLeftCollapsed: (v) => set({ leftCollapsed: v }),
  toggleRight: () => set((s) => ({ rightOpen: !s.rightOpen })),
  setRightOpen: (v) => set({ rightOpen: v }),
  toggleDock: () => set((s) => ({ dockOpen: !s.dockOpen })),
  setDockOpen: (v) => set({ dockOpen: v }),

  setLeftView: (v) => set({ leftCollapsed: false, leftView: v }),
  openDockTab: (tab) => set({ dockOpen: true, dockTab: tab }),
  setDockTab: (tab) => set({ dockTab: tab }),
  openConvInDock: (convId) => set({ dockOpen: true, dockTab: "chat", dockConvId: convId }),
  setDockConvId: (convId) => set({ dockConvId: convId }),
  setView: (v) => set({ view: v }),

  setSelectedCat: (catId) => set({ selectedCat: catId }),
  toggleRecent: () => set((s) => ({ recentOpen: !s.recentOpen })),
  toggleCats: () => set((s) => ({ catsOpen: !s.catsOpen })),
  setLibFilter: (f) => set({ libFilter: f }),
  setArchiveFilter: (f) => set({ archiveFilter: f }),

  toggleNotif: () => set((s) => ({ notifOpen: !s.notifOpen, notifSeen: true })),
  closeNotif: () => set({ notifOpen: false }),
  toggleMute: () => set((s) => ({ notifMuted: !s.notifMuted })),

  toggleMapNode: (id) => set((s) => ({ mapExpanded: { ...s.mapExpanded, [id]: !s.mapExpanded[id] } })),
  toggleMapAll: () => set((s) => ({ mapAllExpanded: !s.mapAllExpanded })),
  mapZoomIn: () => set((s) => ({ mapZoom: Math.min(2, +(s.mapZoom + 0.1).toFixed(2)) })),
  mapZoomOut: () => set((s) => ({ mapZoom: Math.max(0.4, +(s.mapZoom - 0.1).toFixed(2)) })),
  mapZoomReset: () => set({ mapZoom: 1 }),
}));
