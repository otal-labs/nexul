import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
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
  render(
    <QueryClientProvider client={client}>
      <InstanceVersionSection />
    </QueryClientProvider>,
  );
  return client;
};

const upgrading = (status: "pending" | "started") => ({
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
      status,
      error: "",
      requested_by: "user-1",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
  },
});

// DeployStepRow names each phase's state for screen readers in a visually hidden span after the label.
const phaseState = (label: string) => screen.getByText(label).nextElementSibling?.textContent;

const failedUpgrade = (error: string) => ({
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
      error,
      requested_by: "user-1",
      created_at: "2026-09-15T00:00:00Z",
      updated_at: "2026-09-15T00:00:02Z",
    },
  },
});

describe("InstanceVersionSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    confirmOpen.mockReset();
  });

  it("leads with the newer release and offers Upgrade when one is available", async () => {
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

    expect(await screen.findByText("v0.2.0-beta-004 is available")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Release notes for v0.2.0-beta-004" })).toHaveAttribute("href", latest.url);
    expect(screen.getByRole("button", { name: "Upgrade to v0.2.0-beta-004" })).toBeEnabled();
  });

  it("says up to date and offers no Upgrade button when already on the newest release", async () => {
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

    expect(await screen.findByText("Up to date")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^upgrade/i })).not.toBeInTheDocument();
    expect(screen.queryByText("already on the newest release")).not.toBeInTheDocument();
  });

  it("explains why an available release cannot be installed yet, without an Upgrade button", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0-beta-003",
        channel: "beta",
        latest,
        update_available: true,
        can_upgrade: false,
        reason: "instance runner is not connected",
        upgrade: null,
      },
    });
    renderSection();

    expect(await screen.findByText("v0.2.0-beta-004 is available")).toBeInTheDocument();
    expect(screen.getByText("Can't upgrade yet: instance runner is not connected.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^upgrade/i })).not.toBeInTheDocument();
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

    await screen.findByText("This build can't upgrade itself.");
    expect(screen.getByText("No release to compare against")).toBeInTheDocument();
    expect(screen.queryByText("Up to date")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^upgrade/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /release notes/i })).not.toBeInTheDocument();
  });

  it("ticks off the hand-off and runs the install once the upgrade has started", async () => {
    mocks.get.mockResolvedValue(upgrading("started"));
    renderSection();

    await screen.findByText("Download and install v0.2.0-beta-004");
    expect(phaseState("Hand off to this machine")).toBe("done");
    expect(phaseState("Download and install v0.2.0-beta-004")).toBe("active");
    expect(phaseState("Restart on v0.2.0-beta-004")).toBe("pending");
    expect(screen.getByText("Upgrading to v0.2.0-beta-004")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("runs the hand-off while the upgrade is only pending", async () => {
    mocks.get.mockResolvedValue(upgrading("pending"));
    renderSection();

    await screen.findByText("Hand off to this machine");
    expect(phaseState("Hand off to this machine")).toBe("active");
    expect(phaseState("Download and install v0.2.0-beta-004")).toBe("pending");
  });

  it("reads a failed poll mid-upgrade as the restart, not an error", async () => {
    mocks.get.mockResolvedValueOnce(upgrading("started"));
    const client = renderSection();
    await screen.findByText("Download and install v0.2.0-beta-004");

    mocks.get.mockRejectedValueOnce(new Error("Network Error"));
    await client.refetchQueries();

    await waitFor(() => expect(phaseState("Restart on v0.2.0-beta-004")).toBe("active"));
    expect(phaseState("Download and install v0.2.0-beta-004")).toBe("done");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("shows when the last upgrade finished and offers the next one when a newer release exists", async () => {
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

    expect(await screen.findByText(/Upgraded from/)).toHaveTextContent("Upgraded from v0.2.0-beta-003 · 5m ago");
    expect(screen.getByRole("button", { name: "Upgrade to v0.2.0-beta-005" })).toBeEnabled();
  });

  it("fails the install phase with the upgrade's own error and offers to try again", async () => {
    mocks.get.mockResolvedValue(
      failedUpgrade("nexul-server: download https://example.com/nexul-server-linux-arm64: 500 Internal Server Error"),
    );
    renderSection();

    expect(await screen.findByRole("alert")).toHaveTextContent("500 Internal Server Error");
    expect(screen.getByText("Upgrade to v0.2.0-beta-004 failed", { selector: "span" })).toBeInTheDocument();
    expect(phaseState("Hand off to this machine")).toBe("done");
    expect(phaseState("Download and install v0.2.0-beta-004")).toBe("failed");
    expect(screen.getByText("2s")).toBeInTheDocument();
    expect(screen.getByText("journalctl -u 'nexul-upgrade-*'")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeEnabled();
  });

  it("drops an old failure once the instance runs that release anyway", async () => {
    const failed = failedUpgrade("instance is still on v0.2.0-beta-003");
    mocks.get.mockResolvedValue({
      data: { ...failed.data, version: "v0.2.0-beta-004", update_available: false, can_upgrade: false, reason: "already on the newest release" },
    });
    renderSection();

    expect(await screen.findByText("Up to date")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.queryByText("Download and install v0.2.0-beta-004")).not.toBeInTheDocument();
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

  it("checks again past the caches and swaps the answer to the newer release", async () => {
    const onBeta9 = { version: "v0.2.0-beta.9", channel: "beta", update_available: false, can_upgrade: false, reason: "already on the newest release", upgrade: null };
    mocks.get.mockImplementation(async (_url: string, config?: { params?: { refresh?: number } }) =>
      config?.params?.refresh === 1
        ? { data: { ...onBeta9, latest: { version: "v0.2.0-beta.10", url: "u" }, update_available: true, can_upgrade: true, reason: "" } }
        : { data: { ...onBeta9, latest: { version: "v0.2.0-beta.9", url: "u" } } },
    );
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByText("Up to date")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Check again" }));

    expect(await screen.findByRole("button", { name: "Upgrade to v0.2.0-beta.10" })).toBeEnabled();
    expect(screen.getByText("v0.2.0-beta.10 is available")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith("/api/instance/upgrade", { params: { refresh: 1 } });
  });
});
