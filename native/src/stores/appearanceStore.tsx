import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { Uniwind } from "uniwind";

import { kvStateStorage } from "@/lib/storage";

export type Appearance = "system" | "light" | "dark";

export const appearanceLabel: Record<Appearance, string> = { system: "System", light: "Light", dark: "Dark" };

export type AppearanceStore = {
  appearance: Appearance;
  setAppearance: (appearance: Appearance) => void;
};

export const useAppearanceStore = create<AppearanceStore>()(
  persist(
    (set) => ({
      appearance: "system",
      setAppearance: (appearance) => {
        Uniwind.setTheme(appearance);
        set({ appearance });
      },
    }),
    { name: "appearance", storage: createJSONStorage(() => kvStateStorage) },
  ),
);

// The synchronous kv-store hydrates the persisted choice above before this line runs, so a past "light"/"dark" pick reaches Uniwind before first paint.
Uniwind.setTheme(useAppearanceStore.getState().appearance);
