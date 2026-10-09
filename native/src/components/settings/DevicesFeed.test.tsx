import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";
import { DevicesFeed } from "@/components/settings/DevicesFeed";
import type { Session } from "@/models/User";

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ useRouter: () => ({ push: mockPush }) }));

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

beforeEach(() => mockPush.mockReset());

describe("DevicesFeed", () => {
  test("shows the current device first, then the others", async () => {
    await renderFeed([other, current]);

    const rows = await screen.findAllByText(/Android · Pixel 9|Chrome · Desktop/);
    expect(rows.map((node) => node.props.children)).toEqual(["Android · Pixel 9", "Chrome · Desktop"]);
    expect(screen.getByText("This phone")).toBeTruthy();
  });

  test("shows the empty state when there is nothing else signed in", async () => {
    await renderFeed([current]);

    expect(await screen.findByText("No other devices are signed in.")).toBeTruthy();
  });

  test("signing out one device opens the confirm sheet naming that device", async () => {
    await renderFeed([current, other]);

    await userEvent.setup().press(screen.getByLabelText(/^Sign out /));

    expect(mockPush).toHaveBeenCalledWith({
      pathname: "/more/settings/sign-out",
      params: { mode: "device", id: "s-2", label: "Chrome · Desktop" },
    });
  });

  test("sign out everywhere else is disabled with no other devices", async () => {
    await renderFeed([current]);

    expect(screen.getByRole("button", { name: "Sign out everywhere else" })).toBeDisabled();
  });
});
