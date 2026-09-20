import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InstanceVersionSection } from "@/components/settings/InstanceVersionSection";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  errorMessage: vi.fn(() => "error"),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const confirmOpen = vi.fn();
vi.mock("@/hooks/useConfirmationDialog", () => ({
  useConfirmationDialog: () => ({ open: confirmOpen }),
}));

const latest = { version: "v0.2.0-beta-004", url: "https://github.com/otal-labs/nexul/releases/tag/v0.2.0-beta-004" };

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <InstanceVersionSection />
    </QueryClientProvider>,
  );
};

describe("InstanceVersionSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    confirmOpen.mockReset();
  });

  it("shows an enabled upgrade button when a newer release is available", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0-beta-003",
        channel: "beta",
        latest,
        update_available: true,
        can_upgrade: true,
        reason: "",
        upgrade: null,
      },
    });
    renderSection();

    expect(await screen.findByText("v0.2.0-beta-003")).toBeInTheDocument();
    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "v0.2.0-beta-004" })).toHaveAttribute("href", latest.url);
    const button = screen.getByRole("button", { name: "Upgrade to v0.2.0-beta-004" });
    expect(button).toBeEnabled();
  });

  it("disables the button and shows the reason when can_upgrade is false", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0-beta-004",
        channel: "beta",
        latest,
        update_available: false,
        can_upgrade: false,
        reason: "already on the newest release",
        upgrade: null,
      },
    });
    renderSection();

    const button = await screen.findByRole("button", { name: "Upgrade to v0.2.0-beta-004" });
    expect(button).toBeDisabled();
    expect(screen.getByText("already on the newest release")).toBeInTheDocument();
  });

  it("shows friendlier copy for a dev build", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "dev",
        channel: "dev",
        latest: null,
        update_available: false,
        can_upgrade: false,
        reason: "dev build",
        upgrade: null,
      },
    });
    renderSection();

    await screen.findByText("This build cannot upgrade itself.");
    expect(screen.getByRole("button", { name: "Upgrade" })).toBeDisabled();
  });

  it("shows a status line and no button while the upgrade is pending or started", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0-beta-003",
        channel: "beta",
        latest,
        update_available: true,
        can_upgrade: false,
        reason: "an upgrade is already in progress",
        upgrade: {
          id: "u-1",
          from_version: "v0.2.0-beta-003",
          to_version: "v0.2.0-beta-004",
          status: "started",
          error: "",
          requested_by: "user-1",
          created_at: "2026-09-15T00:00:00Z",
          updated_at: "2026-09-15T00:00:00Z",
        },
      },
    });
    renderSection();

    await screen.findByText("Upgrading to v0.2.0-beta-004… the app will reconnect on its own");
    expect(screen.queryByRole("button", { name: /upgrade/i })).not.toBeInTheDocument();
  });

  it("shows the completed message and re-enables the button when a newer release exists", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0-beta-004",
        channel: "beta",
        latest: { version: "v0.2.0-beta-005", url: "https://example.com/v5" },
        update_available: true,
        can_upgrade: true,
        reason: "",
        upgrade: {
          id: "u-1",
          from_version: "v0.2.0-beta-003",
          to_version: "v0.2.0-beta-004",
          status: "completed",
          error: "",
          requested_by: "user-1",
          created_at: "2026-09-15T00:00:00Z",
          updated_at: new Date(Date.now() - 5 * 60_000).toISOString(),
        },
      },
    });
    renderSection();

    expect(await screen.findByText(/Upgraded to v0.2.0-beta-004 at/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Upgrade to v0.2.0-beta-005" })).toBeEnabled();
  });

  it("shows the error and the docker logs hint when the upgrade failed", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0-beta-003",
        channel: "beta",
        latest,
        update_available: true,
        can_upgrade: true,
        reason: "",
        upgrade: {
          id: "u-1",
          from_version: "v0.2.0-beta-003",
          to_version: "v0.2.0-beta-004",
          status: "failed",
          error: "instance is still on v0.2.0-beta-003; run docker logs nexul-upgrade on the host",
          requested_by: "user-1",
          created_at: "2026-09-15T00:00:00Z",
          updated_at: "2026-09-15T00:00:00Z",
        },
      },
    });
    renderSection();

    await screen.findByText(/instance is still on v0.2.0-beta-003/);
    expect(screen.getByText("docker logs nexul-upgrade")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Upgrade to v0.2.0-beta-004" })).toBeEnabled();
  });

  it("posts the upgrade request once the confirmation dialog is accepted", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0-beta-003",
        channel: "beta",
        latest,
        update_available: true,
        can_upgrade: true,
        reason: "",
        upgrade: null,
      },
    });
    mocks.post.mockResolvedValue({
      data: { id: "u-1", from_version: "v0.2.0-beta-003", to_version: "v0.2.0-beta-004", status: "pending", error: "", requested_by: "user-1", created_at: "now", updated_at: "now" },
    });
    confirmOpen.mockResolvedValueOnce(true);
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Upgrade to v0.2.0-beta-004" }));

    expect(confirmOpen).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Upgrade to v0.2.0-beta-004?", confirmLabel: "Upgrade", destructive: false }),
    );
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/upgrade");
  });

  it("does not post when the confirmation dialog is declined", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0-beta-003",
        channel: "beta",
        latest,
        update_available: true,
        can_upgrade: true,
        reason: "",
        upgrade: null,
      },
    });
    confirmOpen.mockResolvedValueOnce(false);
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Upgrade to v0.2.0-beta-004" }));

    expect(mocks.post).not.toHaveBeenCalled();
  });
});
