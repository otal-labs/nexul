import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { kvStateStorage } from "@/lib/storage";

const mockRows = new Map<string, string>();

jest.mock("expo-sqlite/kv-store", () => ({
  __esModule: true,
  default: {
    getItemSync: (key: string) => mockRows.get(key) ?? null,
    setItemSync: (key: string, value: string) => void mockRows.set(key, value),
    removeItemSync: (key: string) => void mockRows.delete(key),
  },
}));

type CounterStore = { count: number; increment: () => void };

const createCounterStore = () =>
  create<CounterStore>()(
    persist(
      (set) => ({ count: 0, increment: () => set((s) => ({ count: s.count + 1 })) }),
      { name: "counter", storage: createJSONStorage(() => kvStateStorage) },
    ),
  );

describe("kvStateStorage", () => {
  beforeEach(() => mockRows.clear());

  test("hydrates a persisted store synchronously on creation", () => {
    createCounterStore().getState().increment();

    expect(createCounterStore().getState().count).toBe(1);
  });
});
