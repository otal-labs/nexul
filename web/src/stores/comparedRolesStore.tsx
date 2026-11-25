import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ComparedRolesStore = {
  // The roles left out of Compare roles, so a role made later shows by default; ids of deleted roles are left to the matrix to skip.
  hidden: string[];
  toggle: (roleId: string) => void;
};

export const useComparedRolesStore = create<ComparedRolesStore>()(
  persist(
    (set) => ({
      hidden: [],
      toggle: (roleId) => set((s) => ({ hidden: s.hidden.includes(roleId) ? s.hidden.filter((id) => id !== roleId) : [...s.hidden, roleId] })),
    }),
    { name: "compared-roles-hidden", partialize: (s) => ({ hidden: s.hidden }) },
  ),
);

// The roles left shown, in the order given; every role when none is, so the matrix never comes up empty.
export const shownRoles = <T extends { id: string }>(roles: T[], hidden: string[]): T[] => {
  const shown = roles.filter((role) => !hidden.includes(role.id));
  return shown.length > 0 ? shown : roles;
};
