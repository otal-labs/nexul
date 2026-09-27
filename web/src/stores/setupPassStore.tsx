import { create } from "zustand";
import { persist } from "zustand/middleware";

import type { SetupPass } from "@/models/Setup";

export type SetupPassStore = {
  token: string | null;
  expiresAt: string | null;
  // Memory only: lets the handoff link carry the code to the domain without it ever touching storage.
  code: string | null;
  unlock: (pass: SetupPass, code: string) => void;
  clear: () => void;
};

// The first-run bearer from POST /api/setup/unlock; only valid while no user exists, never a session.
export const useSetupPassStore = create<SetupPassStore>()(
  persist(
    (set) => ({
      token: null,
      expiresAt: null,
      code: null,
      unlock: (pass, code) => set({ token: pass.token, expiresAt: pass.expires_at, code }),
      clear: () => set({ token: null, expiresAt: null }),
    }),
    {
      name: "setup-pass",
      partialize: (state) => ({ token: state.token, expiresAt: state.expiresAt }),
    },
  ),
);

// Drops an expired pass on read, so the next render falls back to the code screen instead of a round of 401s.
export const currentSetupPass = (now = Date.now()): string | null => {
  const { token, expiresAt, clear } = useSetupPassStore.getState();
  if (!token) return null;
  if (expiresAt && Date.parse(expiresAt) <= now) {
    clear();
    return null;
  }
  return token;
};
