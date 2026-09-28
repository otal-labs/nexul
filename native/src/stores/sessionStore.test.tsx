import { queryClient } from "@/lib/queryClient";
import { clearPushToken } from "@/push/pushToken";
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

jest.mock("@/push/pushToken", () => ({
  registerPushToken: jest.fn(async () => undefined),
  clearPushToken: jest.fn(async () => undefined),
}));

const mockSecrets = (jest.requireMock("expo-secure-store") as { secrets: Map<string, string> }).secrets;

describe("sessionStore", () => {
  beforeEach(() => {
    mockSecrets.clear();
    jest.clearAllMocks();
    useSessionStore.setState({ host: null, signedIn: false });
  });

  test("starts signed out when the secure store is empty", () => {
    expect(useSessionStore.getState().signedIn).toBe(false);
    expect(readSessionToken()).toBeNull();
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

  test("signing out clears the push token while the session can still authenticate", () => {
    useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");

    useSessionStore.getState().signOut();

    expect(clearPushToken).toHaveBeenCalledWith("https://nexul.example.com", "ses_abc");
  });

  test("signing out while already signed out never calls the server", () => {
    useSessionStore.getState().signOut();

    expect(clearPushToken).not.toHaveBeenCalled();
  });
});
