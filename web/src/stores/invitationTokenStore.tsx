import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type InvitationTokenStore = {
  token: string | null;
  keep: (token: string) => void;
  clear: () => void;
};

// The invite token survives the provider round trip in this tab only, so "Not you?" can start another provider's sign-in.
export const useInvitationTokenStore = create<InvitationTokenStore>()(
  persist(
    (set) => ({
      token: null,
      keep: (token) => set({ token }),
      clear: () => set({ token: null }),
    }),
    { name: "invitation-token", storage: createJSONStorage(() => sessionStorage) },
  ),
);
