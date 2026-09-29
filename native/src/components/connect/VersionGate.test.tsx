import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent, waitFor } from "@testing-library/react-native";
import { Text } from "react-native";

import { api } from "@/api/client";
import { fetchAbout } from "@/api/connect";
import { VersionGate } from "@/components/connect/VersionGate";
import { useSessionStore } from "@/stores/sessionStore";

jest.mock("@/api/client", () => ({ api: { delete: jest.fn() } }));
jest.mock("@/api/connect", () => ({ fetchAbout: jest.fn() }));
jest.mock("@/push/pushToken", () => ({ clearPushToken: jest.fn(async () => undefined) }));

jest.mock("expo-secure-store", () => {
  const secrets = new Map<string, string>();
  return {
    getItem: (key: string) => secrets.get(key) ?? null,
    setItem: (key: string, value: string) => void secrets.set(key, value),
    deleteItemAsync: async (key: string) => void secrets.delete(key),
  };
});

beforeEach(() => {
  jest.mocked(fetchAbout).mockResolvedValue({ product: "nexul", version: "v0.1.0" });
  jest.mocked(api.delete).mockRejectedValue(new Error("offline"));
  useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");
});

describe("VersionGate", () => {
  test("a signed-in phone refused by an old server can sign out instead of only retrying", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    await render(
      <QueryClientProvider client={client}>
        <VersionGate>
          <Text>Connect</Text>
        </VersionGate>
      </QueryClientProvider>,
    );

    const user = userEvent.setup();
    await user.press(await screen.findByRole("button", { name: "Sign out" }));
    await user.press(screen.getByRole("button", { name: "Confirm sign out" }));

    await waitFor(() => expect(useSessionStore.getState().signedIn).toBe(false));
    expect(await screen.findByText("Connect")).toBeTruthy();
  });
});
