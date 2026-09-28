import { renderHook } from "@testing-library/react-native";

import { registerPushToken } from "@/push/pushToken";
import { usePushRegistration } from "@/push/usePushRegistration";
import { useSessionStore } from "@/stores/sessionStore";

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

describe("usePushRegistration", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useSessionStore.setState({ host: null, signedIn: false });
  });

  test("registers the token after sign-in", async () => {
    useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");

    await renderHook(() => usePushRegistration());

    expect(registerPushToken).toHaveBeenCalledWith("https://nexul.example.com", "ses_abc");
  });

  test("does nothing while signed out", async () => {
    await renderHook(() => usePushRegistration());

    expect(registerPushToken).not.toHaveBeenCalled();
  });
});
