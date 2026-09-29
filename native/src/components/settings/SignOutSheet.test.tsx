import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent, waitFor } from "@testing-library/react-native";

import { ApiError, api } from "@/api/client";
import { SignOutSheet } from "@/components/settings/SignOutSheet";

jest.mock("@/api/client", () => ({
  ...jest.requireActual("@/api/client"),
  api: { delete: jest.fn() },
}));

// SessionHooks pulls in the real queryClient (via sessionStore), which subscribes to this on mount; unmocked, its subscription lacks .remove() under jest.
jest.mock("expo-network", () => ({
  addNetworkStateListener: () => ({ remove: () => undefined }),
  getNetworkStateAsync: () => Promise.resolve({ isConnected: true }),
}));

jest.mock("expo-secure-store", () => ({
  getItem: (key: string) => (key === "session_host" ? "https://nexul.example.com" : null),
  setItem: jest.fn(),
  deleteItemAsync: jest.fn(),
}));

const mockBack = jest.fn();
let mockParams: Record<string, string> = {};
jest.mock("expo-router", () => ({
  useRouter: () => ({ back: mockBack }),
  useLocalSearchParams: () => mockParams,
}));

const renderSheet = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <SignOutSheet />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  jest.mocked(api.delete).mockReset().mockResolvedValue(undefined);
  mockBack.mockReset();
});

describe("SignOutSheet", () => {
  test("names the other device and deletes only its session", async () => {
    mockParams = { mode: "device", id: "s-2", label: "Chrome · Desktop" };
    await renderSheet();

    expect(screen.getByText("Sign out Chrome · Desktop?")).toBeTruthy();
    await userEvent.setup().press(screen.getByRole("button", { name: "Sign out" }));

    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/auth/sessions/s-2"));
    await waitFor(() => expect(mockBack).toHaveBeenCalled());
  });

  test("signing out of this phone names the instance", async () => {
    mockParams = { mode: "current" };
    await renderSheet();

    expect(screen.getByText("Sign out of nexul.example.com?")).toBeTruthy();
  });

  test("signing out everywhere else deletes every other session at once", async () => {
    mockParams = { mode: "others" };
    await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Sign out" }));

    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/auth/sessions/others"));
  });

  test("a failed sign out keeps the sheet open and shows the error", async () => {
    mockParams = { mode: "others" };
    jest.mocked(api.delete).mockRejectedValue(new ApiError(500, { message: "server down", code: "INTERNAL" }, "DELETE failed: 500"));
    await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Sign out" }));

    expect(await screen.findByText(/server down/)).toBeTruthy();
    expect(mockBack).not.toHaveBeenCalled();
  });

  test("Cancel dismisses without signing anyone out", async () => {
    mockParams = { mode: "others" };
    await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Cancel" }));

    expect(mockBack).toHaveBeenCalled();
    expect(api.delete).not.toHaveBeenCalled();
  });
});
