import { create } from "zustand";
import { persist } from "zustand/middleware";

export type PhoneBannerStore = {
  dismissed: boolean;
  dismiss: () => void;
};

export const usePhoneBannerStore = create<PhoneBannerStore>()(
  persist(
    (set) => ({
      dismissed: false,
      dismiss: () => set({ dismissed: true }),
    }),
    { name: "phone-banner", partialize: (s) => ({ dismissed: s.dismissed }) },
  ),
);
