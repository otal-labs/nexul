import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { StackWatchButton } from "@/components/stack/StackWatchButton";
import type { StackWithBranches } from "@/models/Stack";
import { resetViewer, viewer } from "@/test/viewerActions";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(() => "boom"),
}));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);

const stack = (over: Partial<StackWithBranches> = {}): StackWithBranches => ({
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "instance",
  strategy: "run",
  managed: false,
  created_at: "2026-08-12T09:00:00Z",
  updated_at: "2026-08-12T09:00:00Z",
  watching: true,
  ...over,
});

const serve = (served: StackWithBranches) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/stacks/stack-1") return { data: served };
    return { data: { effective: [] } };
  });

const renderButton = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <StackWatchButton stackId="stack-1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.delete).mockReset();
  resetViewer();
  viewer.workspace = ["stacks:read"];
});

describe("StackWatchButton", () => {
  it("is pressed while the viewer watches, and pressing it stops watching at once", async () => {
    serve(stack());
    vi.mocked(api.delete).mockResolvedValue({ data: { watching: false } });
    const user = userEvent.setup();
    renderButton();

    const button = await screen.findByRole("button", { name: "Watching" });
    expect(button).toHaveAttribute("aria-pressed", "true");
    await user.click(button);

    expect(api.delete).toHaveBeenCalledWith("/api/stacks/stack-1/watchers/me");
    const off = await screen.findByRole("button", { name: "Watch" });
    expect(off).toHaveAttribute("aria-pressed", "false");
  });

  it("turns watching back on", async () => {
    serve(stack({ watching: false }));
    vi.mocked(api.put).mockResolvedValue({ data: { watching: true } });
    const user = userEvent.setup();
    renderButton();

    await user.click(await screen.findByRole("button", { name: "Watch" }));

    expect(api.put).toHaveBeenCalledWith("/api/stacks/stack-1/watchers/me");
    expect(await screen.findByRole("button", { name: "Watching" })).toHaveAttribute("aria-pressed", "true");
  });

  it("goes back to what the server holds when the change is refused", async () => {
    serve(stack());
    vi.mocked(api.delete).mockRejectedValue(new Error("nope"));
    const user = userEvent.setup();
    renderButton();

    await user.click(await screen.findByRole("button", { name: "Watching" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Watching" })).toHaveAttribute("aria-pressed", "true"));
  });

  it("is absent on a stack outside every project", async () => {
    serve(stack({ project_id: "" }));
    renderButton();

    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/stacks/stack-1"));
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
