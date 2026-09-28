import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent, waitFor } from "@testing-library/react-native";
import { Alert } from "react-native";

import { api } from "@/api/client";
import { DevicesFeed } from "@/components/settings/DevicesFeed";
import type { Session } from "@/models/User";

jest.mock("@/api/client", () => ({
  api: { delete: jest.fn() },
}));

// SessionHooks pulls in the real queryClient (via sessionStore), which subscribes to this on mount; unmocked, its subscription lacks .remove() under jest.
jest.mock("expo-network", () => ({
  addNetworkStateListener: () => ({ remove: () => undefined }),
  getNetworkStateAsync: () => Promise.resolve({ isConnected: true }),
}));

const current: Session = { id: "s-1", client: "phone", platform: "Android", label: "Pixel 9", ip: "10.0.0.1", last_active_at: "2026-08-12T12:00:00Z", current: true };
const other: Session = { id: "s-2", client: "browser", platform: "Chrome", label: "Desktop", ip: "10.0.0.2", last_active_at: "2026-08-12T11:00:00Z", current: false };

const renderFeed = (sessions: Session[]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DevicesFeed sessions={sessions} />
    </QueryClientProvider>,
  );
};

// Confirms are native Alerts; pressing confirms them by invoking the destructive button's onPress directly.
const confirm = () => {
  const call = jest.mocked(Alert.alert).mock.calls.at(-1);
  const buttons = call?.[2] as { text: string; onPress?: () => void }[];
  buttons.find((button) => button.text !== "Cancel")?.onPress?.();
};

beforeEach(() => {
  jest.mocked(api.delete).mockReset().mockResolvedValue(undefined);
  jest.spyOn(Alert, "alert").mockImplementation(() => undefined);
});

describe("DevicesFeed", () => {
  test("shows the current device first, then the others", async () => {
    await renderFeed([current, other]);

    const rows = await screen.findAllByText(/Android · Pixel 9|Chrome · Desktop/);
    expect(rows.map((node) => node.props.children)).toEqual(["Android · Pixel 9", "Chrome · Desktop"]);
    expect(screen.getByText("This device")).toBeTruthy();
  });

  test("shows the empty state when there is nothing else signed in", async () => {
    await renderFeed([current]);

    expect(await screen.findByText("No other devices are signed in.")).toBeTruthy();
  });

  test("signing out one device deletes its session", async () => {
    await renderFeed([current, other]);

    await userEvent.setup().press(screen.getByLabelText("Sign out"));
    confirm();

    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/auth/sessions/s-2"));
  });

  test("signing out everywhere else deletes every other session at once", async () => {
    await renderFeed([current, other]);

    await userEvent.setup().press(screen.getByRole("button", { name: "Sign out everywhere else" }));
    confirm();

    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/auth/sessions/others"));
  });

  test("sign out everywhere else is disabled with no other devices", async () => {
    await renderFeed([current]);

    expect(screen.getByRole("button", { name: "Sign out everywhere else" })).toBeDisabled();
  });
});
