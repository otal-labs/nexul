import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DevicesFeed } from "@/components/you/DevicesFeed";
import type { Session } from "@/models/User";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const session = (overrides: Partial<Session>): Session => ({
  id: "s1",
  user_id: "u1",
  client: "browser",
  platform: "Linux",
  label: "Chrome",
  ip: "10.0.0.1",
  created_at: "2026-09-01T00:00:00Z",
  last_active_at: "2026-09-01T00:00:00Z",
  expires_at: "2026-10-01T00:00:00Z",
  current: false,
  ...overrides,
});

const current = session({ id: "s-current", platform: "Linux", label: "Firefox", current: true });
const phone = session({ id: "s-phone", client: "phone", platform: "Android", label: "Pixel 8", ip: "82.14.201.9" });
const laptop = session({ id: "s-laptop", platform: "macOS", label: "Nexul desktop" });

const renderFeed = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DevicesFeed />
    </QueryClientProvider>,
  );
};

const armAndConfirm = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(await screen.findByRole("button", { name: "Sign out" }));
  await user.click(screen.getByRole("button", { name: "Sign out" }));
};

describe("DevicesFeed", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockReset();
    vi.mocked(toast.error).mockClear();
    useDeviceArrivalStore.setState({ arrivals: [] });
  });

  it("keeps a phone that just connected signable out while it glows", async () => {
    useDeviceArrivalStore.setState({ arrivals: [{ id: "s-phone", platform: "Android", label: "Pixel 8", at: Date.now() + 1_000 }] });
    mocks.get.mockResolvedValue({ data: { sessions: [current, phone] } });
    renderFeed();

    expect(await screen.findByText("Android · Pixel 8")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign out" })).toBeEnabled();
  });

  it("shows an error when the list fails to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Sessions failed");
    renderFeed();
    expect(await screen.findByText("Sessions failed")).toBeInTheDocument();
  });

  it("lists the current device first, then the others", async () => {
    mocks.get.mockResolvedValue({ data: { sessions: [phone, current] } });
    renderFeed();
    await screen.findByText("Android · Pixel 8");
    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("Linux · Firefox");
    expect(rows[0]).toHaveTextContent("This device");
    expect(rows[1]).toHaveTextContent("Android · Pixel 8");
    expect(rows[1]).toHaveTextContent("82.14.201.9");
    expect(screen.getByRole("button", { name: "Sign out everywhere else" })).toBeEnabled();
  });

  it("shows an empty row and disables sign out everywhere when only this device is signed in", async () => {
    mocks.get.mockResolvedValue({ data: { sessions: [current] } });
    renderFeed();
    expect(await screen.findByText(/no other devices are signed in/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign out everywhere else" })).toBeDisabled();
  });

  it("signs a device out and removes its row once the list refetches", async () => {
    mocks.get
      .mockResolvedValueOnce({ data: { sessions: [current, phone] } })
      .mockResolvedValue({ data: { sessions: [current] } });
    mocks.del.mockResolvedValue({});
    const user = userEvent.setup();
    renderFeed();

    await armAndConfirm(user);

    expect(mocks.del).toHaveBeenCalledWith("/api/auth/sessions/s-phone");
    await waitFor(() => expect(screen.queryByText("Android · Pixel 8")).not.toBeInTheDocument());
    expect(screen.getByText(/no other devices are signed in/i)).toBeInTheDocument();
  });

  it("keeps the row and shows an error when signing out fails", async () => {
    mocks.get.mockResolvedValue({ data: { sessions: [current, phone] } });
    mocks.del.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Could not sign out");
    const user = userEvent.setup();
    renderFeed();

    await armAndConfirm(user);

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Could not sign out"));
    expect(screen.getByText("Android · Pixel 8")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign out" })).toBeEnabled();
  });

  it("signs out everywhere else and leaves only this device", async () => {
    mocks.get
      .mockResolvedValueOnce({ data: { sessions: [current, phone, laptop] } })
      .mockResolvedValue({ data: { sessions: [current] } });
    mocks.del.mockResolvedValue({});
    const user = userEvent.setup();
    renderFeed();

    await user.click(await screen.findByRole("button", { name: "Sign out everywhere else" }));

    expect(mocks.del).toHaveBeenCalledWith("/api/auth/sessions/others");
    expect(await screen.findByText(/no other devices are signed in/i)).toBeInTheDocument();
    expect(screen.queryByText("macOS · Nexul desktop")).not.toBeInTheDocument();
    expect(screen.getByText("Linux · Firefox")).toBeInTheDocument();
  });
});
