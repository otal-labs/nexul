import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type InterviewBannerStore = {
  dismissedProjectIds: string[];
  dismiss: (projectId: string) => void;
};

// sessionStorage, not localStorage: a dismissal lasts one tab session, so the banner returns until the interview exists.
export const useInterviewBannerStore = create<InterviewBannerStore>()(
  persist(
    (set) => ({
      dismissedProjectIds: [],
      dismiss: (projectId) =>
        set((s) => ({ dismissedProjectIds: [...s.dismissedProjectIds.filter((id) => id !== projectId), projectId] })),
    }),
    { name: "interview-banner", storage: createJSONStorage(() => sessionStorage) },
  ),
);
