import { queryClient } from "@/lib/queryClient";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// The store reads the secure store while its module loads, so the fake owns its map inside the hoisted factory.
jest.mock("expo-secure-store", () => {
  const secrets = new Map<string, string>();
  return {
    secrets,
    getItem: (key: string) => secrets.get(key) ?? null,
    setItem: (key: string, value: string) => void secrets.set(key, value),
    deleteItemAsync: async (key: string) => void secrets.delete(key),
  };
});

const mockSecrets = (jest.requireMock("expo-secure-store") as { secrets: Map<string, string> }).secrets;

describe("sessionStore", () => {
  beforeEach(() => {
    mockSecrets.clear();
    useSessionStore.setState({ host: null, signedIn: false });
  });

  test("signing in keeps the token in the secure store, never in the store state", () => {
    useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");

    expect(useSessionStore.getState()).toMatchObject({ host: "https://nexul.example.com", signedIn: true });
    expect(Object.values(useSessionStore.getState())).not.toContain("ses_abc");
    expect(readSessionToken()).toBe("ses_abc");
  });

  test("signing out clears the secure store and the query cache", async () => {
    useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");
    queryClient.setQueryData(["getAbout"], { product: "nexul", version: "dev" });

    useSessionStore.getState().signOut();
    await Promise.resolve();

    expect(useSessionStore.getState()).toMatchObject({ host: null, signedIn: false });
    expect(readSessionToken()).toBeNull();
    expect(queryClient.getQueryCache().getAll()).toHaveLength(0);
  });
});
