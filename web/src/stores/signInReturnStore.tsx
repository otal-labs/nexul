import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type SignInReturnStore = {
  path: string | null;
  keep: (path: string) => void;
  clear: () => void;
};

// A page that needs a sign-in first keeps its address here, in this tab only, to come back to across the provider round trip.
export const useSignInReturnStore = create<SignInReturnStore>()(
  persist(
    (set) => ({
      path: null,
      keep: (path) => set({ path }),
      clear: () => set({ path: null }),
    }),
    { name: "sign-in-return", storage: createJSONStorage(() => sessionStorage) },
  ),
);

// takeSignInReturn is where a finished sign-in lands: the kept page once, else home.
export const takeSignInReturn = (): string => {
  const { path, clear } = useSignInReturnStore.getState();
  clear();
  return path ?? "/";
};
