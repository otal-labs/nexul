import { queryClient } from "@/lib/queryClient";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// The store reads the secure store while its module loads, so the fake owns its map inside the hoisted factory.
jest.mock("expo-secure-store", () => {
  const secrets = new Map<string, string>();
  return {
    secrets,
    getItem: jest.fn((key: string) => secrets.get(key) ?? null),
    setItem: (key: string, value: string) => void secrets.set(key, value),
    deleteItemAsync: async (key: string) => void secrets.delete(key),
  };
});

const mockSecureStore = jest.requireMock("expo-secure-store") as { secrets: Map<string, string>; getItem: jest.Mock };
const mockSecrets = mockSecureStore.secrets;

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

  // A secure-store read decrypts through the Android Keystore on the JS thread, and every request needs the token.
  test("reading the token for each request never goes back to the secure store", () => {
    useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");
    mockSecureStore.getItem.mockClear();

    for (let i = 0; i < 10; i++) expect(readSessionToken()).toBe("ses_abc");

    expect(mockSecureStore.getItem).not.toHaveBeenCalled();
  });

  test("a session saved by an earlier launch is read once when the app starts", () => {
    mockSecrets.set("session_host", "https://nexul.example.com");
    mockSecrets.set("session_token", "ses_saved");

    jest.isolateModules(() => {
      const fresh = jest.requireActual<typeof import("@/stores/sessionStore")>("@/stores/sessionStore");
      expect(fresh.useSessionStore.getState()).toMatchObject({ host: "https://nexul.example.com", signedIn: true });
      expect(fresh.readSessionToken()).toBe("ses_saved");
    });
  });
});
