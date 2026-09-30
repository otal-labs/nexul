import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ModelFavouritesStore = {
  // modelKey(providerId, slug) of each starred model.
  favourites: string[];
  toggle: (key: string) => void;
};

export const useModelFavouritesStore = create<ModelFavouritesStore>()(
  persist(
    (set) => ({
      favourites: [],
      toggle: (key) =>
        set((s) => ({ favourites: s.favourites.includes(key) ? s.favourites.filter((k) => k !== key) : [...s.favourites, key] })),
    }),
    { name: "model-favourites", partialize: (s) => ({ favourites: s.favourites }) },
  ),
);
