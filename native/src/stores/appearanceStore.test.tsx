import { useAppearanceStore } from "@/stores/appearanceStore";

jest.mock("expo-sqlite/kv-store", () => {
  const rows = new Map<string, string>();
  return {
    __esModule: true,
    default: {
      getItemSync: (key: string) => rows.get(key) ?? null,
      setItemSync: (key: string, value: string) => void rows.set(key, value),
      removeItemSync: (key: string) => void rows.delete(key),
    },
  };
});

jest.mock("uniwind", () => {
  const setTheme = jest.fn();
  return { Uniwind: { setTheme } };
});

const mockUniwind = jest.requireMock("uniwind") as { Uniwind: { setTheme: jest.Mock } };

describe("appearanceStore", () => {
  beforeEach(() => {
    useAppearanceStore.setState({ appearance: "system" });
    mockUniwind.Uniwind.setTheme.mockClear();
  });

  test("setting a choice applies it through Uniwind and persists it", () => {
    useAppearanceStore.getState().setAppearance("dark");

    expect(mockUniwind.Uniwind.setTheme).toHaveBeenCalledWith("dark");
    expect(useAppearanceStore.getState().appearance).toBe("dark");
  });

  test("persists across a fresh read of the store", () => {
    useAppearanceStore.getState().setAppearance("light");

    const persisted = JSON.parse(
      (jest.requireMock("expo-sqlite/kv-store") as { default: { getItemSync: (k: string) => string | null } }).default.getItemSync(
        "appearance",
      ) ?? "null",
    );

    expect(persisted.state.appearance).toBe("light");
  });
});
