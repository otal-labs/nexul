import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { GitHubAccessSection } from "@/components/you/GitHubAccessSection";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), del: vi.fn(), assign: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, delete: mocks.del },
  errorMessage: (error: unknown) => String(error),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const stub = (link: object) =>
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/auth/github-link") return Promise.resolve({ data: link });
    if (url === "/api/auth/bootstrap-status") return Promise.resolve({ data: { configured: true } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <GitHubAccessSection />
    </QueryClientProvider>,
  );
};

describe("GitHubAccessSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.del.mockReset();
    vi.stubGlobal("location", { ...window.location, assign: mocks.assign });
  });

  it("connects GitHub through the sign-in link flow, so it joins this profile", async () => {
    stub({ state: "none" });
    mocks.post.mockResolvedValue({ data: { url: "https://github.com/login/oauth/authorize?state=link.x" } });
    renderSection();
    const user = userEvent.setup();

    expect(await screen.findByText("Not connected")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Disconnect GitHub" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Connect GitHub" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/identities/link", { provider: "github" });
  });

  it("asks to reconnect after GitHub refused a refresh, and can still disconnect", async () => {
    stub({ state: "reconnect", login: "alice" });
    renderSection();

    expect(await screen.findByText("Reconnect needed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reconnect" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Disconnect GitHub" })).toBeInTheDocument();
  });

  it("disconnects only once confirmed", async () => {
    stub({ state: "connected", login: "alice" });
    mocks.del.mockResolvedValue({});
    renderSection();
    const user = userEvent.setup();

    expect(await screen.findByText("@alice")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Disconnect GitHub" }));
    expect(mocks.del).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Disconnect" }));
    expect(mocks.del).toHaveBeenCalledWith("/api/auth/github-link");
  });
});
