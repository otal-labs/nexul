import { create } from "zustand";

export interface ProjectListing {
  roleIds: string[];
  userIds: string[];
}

// The Access page's held ticks until a save: only the rows touched, each as a whole, over what the server holds.
export type AccessDraftStore = {
  projects: Record<string, ProjectListing>;
  channels: Record<string, string[]>;
  setProjects: (rows: Record<string, ProjectListing>) => void;
  setChannels: (rows: Record<string, string[]>) => void;
  clearProjects: () => void;
  clearChannels: () => void;
  clear: () => void;
};

export const useAccessDraftStore = create<AccessDraftStore>((set) => ({
  projects: {},
  channels: {},
  setProjects: (rows) => set((s) => ({ projects: { ...s.projects, ...rows } })),
  setChannels: (rows) => set((s) => ({ channels: { ...s.channels, ...rows } })),
  clearProjects: () => set({ projects: {} }),
  clearChannels: () => set({ channels: {} }),
  clear: () => set({ projects: {}, channels: {} }),
}));
