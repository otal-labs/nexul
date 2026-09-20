import { create } from "zustand";
import { persist } from "zustand/middleware";

export type SessionStore = {
  token: string | null;
  isLoggedIn: boolean;
  login: (token: string) => void;
  logout: () => void;
};

// The resolved user is server state, never mirrored here (F5); only token/isLoggedIn are client state.
export const useSessionStore = create<SessionStore>()(
  persist(
    (set) => ({
      token: null,
      isLoggedIn: false,
      login: (token) => set({ isLoggedIn: true, token }),
      logout: () => {
        set({ isLoggedIn: false, token: null });
        useSessionStore.persist.clearStorage();
      },
    }),
    {
      name: "session",
      partialize: (state) => ({ token: state.token, isLoggedIn: state.isLoggedIn }),
    },
  ),
);
