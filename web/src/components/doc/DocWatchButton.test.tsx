import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocWatchButton } from "@/components/doc/DocWatchButton";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

const watcher = (userId: string, source: "auto" | "manual") => ({
  user_id: userId,
  source,
  created_at: "2026-10-01T12:00:00Z",
});

const renderButton = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DocWatchButton docId="doc-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: { user: { id: "u-rix" } } };
    return { data: { watchers: [watcher("u-onik", "auto")], watching: false } };
  });
});

describe("DocWatchButton", () => {
  it("watches and stops watching for the viewer, and the count follows", async () => {
    vi.mocked(api.put).mockResolvedValue({
      data: { watchers: [watcher("u-onik", "auto"), watcher("u-rix", "manual")], watching: true },
    });
    vi.mocked(api.delete).mockResolvedValue({ data: { watchers: [watcher("u-onik", "auto")], watching: false } });
    const user = userEvent.setup();
    renderButton();

    await user.click(await screen.findByRole("button", { name: "Watchers: 1" }));
    expect(await screen.findByText("u-onik")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Watch" }));

    expect(api.put).toHaveBeenCalledWith("/api/docs/doc-1/watchers/me");
    expect(await screen.findByRole("button", { name: "Watchers: 2" })).toBeInTheDocument();
    expect(screen.getByText("you")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Stop watching" }));
    expect(api.delete).toHaveBeenCalledWith("/api/docs/doc-1/watchers/me");
    expect(await screen.findByRole("button", { name: "Watchers: 1" })).toBeInTheDocument();
    expect(screen.queryByText("you")).not.toBeInTheDocument();
  });

  it("says so when nobody watches the doc", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: { id: "u-rix" } } };
      return { data: { watchers: [], watching: false } };
    });
    const user = userEvent.setup();
    renderButton();

    await user.click(await screen.findByRole("button", { name: "Watchers: 0" }));
    expect(await screen.findByText("Nobody is watching this doc")).toBeInTheDocument();
  });
});
