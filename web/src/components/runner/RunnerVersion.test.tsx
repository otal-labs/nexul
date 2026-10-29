import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnerVersion } from "@/components/runner/RunnerVersion";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const renderVersion = (version: string) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RunnerVersion version={version} />
    </QueryClientProvider>,
  );

describe("RunnerVersion", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("marks a runner behind the server with an arrow and names the version it updates to", async () => {
    mocks.get.mockResolvedValue({ data: { version: "v0.3.68-beta" } });
    renderVersion("v0.3.61-beta");

    expect(await screen.findByTitle("Updates to v0.3.68-beta on its next connect")).toHaveTextContent("v0.3.61-beta ↑");
  });

  it.each([
    ["a runner on the server's version", "v0.3.68-beta", "v0.3.68-beta"],
    ["a local build", "dev", "v0.3.68-beta"],
    ["a dev server", "v0.3.61-beta", "dev"],
  ])("marks nothing for %s", async (_, version, server) => {
    mocks.get.mockResolvedValue({ data: { version: server } });
    renderVersion(version);

    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/version"));
    expect(screen.getByText(version)).toBeInTheDocument();
    expect(screen.queryByText(/↑/)).not.toBeInTheDocument();
  });

  it("marks nothing when the server version lookup fails", async () => {
    mocks.get.mockRejectedValue(new Error("down"));
    renderVersion("v0.3.61-beta");

    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/version"));
    expect(screen.queryByText(/↑/)).not.toBeInTheDocument();
  });
});
