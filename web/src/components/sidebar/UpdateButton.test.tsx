import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UpdateButton } from "@/components/sidebar/UpdateButton";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const renderButton = (enabled = true) =>
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <UpdateButton enabled={enabled} />
        <span>after</span>
      </QueryClientProvider>
    </MemoryRouter>,
  );

const withUpdate = (changes: unknown[]) => ({
  data: {
    version: "v0.2.0-beta.7",
    channel: "beta",
    latest: { version: "v0.2.0-beta.8", url: "https://github.com/otal-labs/nexul/releases/tag/v0.2.0-beta.8" },
    update_available: true,
    changes,
  },
});

describe("UpdateButton", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("renders nothing when the instance is current", async () => {
    mocks.get.mockResolvedValue({
      data: { version: "v0.2.0", channel: "stable", latest: null, update_available: false, changes: [] },
    });
    renderButton();
    await screen.findByText("after");
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalled());
    expect(screen.queryByRole("link", { name: /Update available/ })).not.toBeInTheDocument();
  });

  it("links to the instance version settings", async () => {
    mocks.get.mockResolvedValue(withUpdate([]));
    renderButton();
    const button = await screen.findByRole("link", { name: "Update available: v0.2.0-beta.8" });
    expect(button).toHaveAttribute("href", "/configuration/instance#instance-version");
  });

  it("hovering shows what changed since the running version", async () => {
    const user = userEvent.setup();
    mocks.get.mockResolvedValue(
      withUpdate([
        { version: "v0.2.0-beta.8", url: "https://x/8", notes: ["Resume the pairing wizard"] },
      ]),
    );
    renderButton();

    await user.hover(await screen.findByRole("link", { name: /Update available/ }));

    expect(await screen.findByText("v0.2.0-beta.7 → v0.2.0-beta.8")).toBeInTheDocument();
    expect(screen.getByText("Resume the pairing wizard")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "v0.2.0-beta.8" })).toHaveAttribute("href", "https://x/8");
    expect(screen.getByRole("link", { name: "Upgrade from Settings" })).toHaveAttribute(
      "href",
      "/configuration/instance#instance-version",
    );
  });

  it("without notes, points at the release page instead", async () => {
    const user = userEvent.setup();
    mocks.get.mockResolvedValue(withUpdate([]));
    renderButton();

    await user.hover(await screen.findByRole("link", { name: /Update available/ }));

    expect(await screen.findByRole("link", { name: "Read the release notes" })).toHaveAttribute(
      "href",
      "https://github.com/otal-labs/nexul/releases/tag/v0.2.0-beta.8",
    );
  });

  it("does not fetch the version when logged out", () => {
    renderButton(false);
    expect(mocks.get).not.toHaveBeenCalled();
  });
});
