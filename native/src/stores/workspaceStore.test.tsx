import { useWorkspaceStore } from "@/stores/workspaceStore";

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

describe("workspaceStore", () => {
  beforeEach(() => useWorkspaceStore.setState({ selectedWorkspaceId: "" }));

  test("starts with no workspace selected", () => {
    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("");
  });

  test("switching workspace updates the store", () => {
    useWorkspaceStore.getState().selectWorkspace("ws-2");

    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-2");
  });
});
