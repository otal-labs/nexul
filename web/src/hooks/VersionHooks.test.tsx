import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { shouldPromptReload, useServerVersion } from "@/hooks/VersionHooks";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const version = {
  version: "v0.2.0-beta-310",
  channel: "beta",
  latest: { version: "v0.2.0-beta-331", url: "https://github.com/otal-labs/nexul/releases/tag/v0.2.0-beta-331" },
  update_available: true,
};

const Harness = ({ enabled = true }: { enabled?: boolean }) => {
  const query = useServerVersion(enabled);
  return (
    <span data-testid="state">
      {query.isPending ? "loading" : query.isSuccess ? "loaded" : "error"}
    </span>
  );
};

describe("useServerVersion", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("fetches the server version", async () => {
    mocks.get.mockResolvedValue({ data: version });
    render(
      <QueryClientProvider client={new QueryClient()}>
        <Harness />
      </QueryClientProvider>,
    );
    expect(mocks.get).toHaveBeenCalledWith("/api/version");
    await screen.findByText("loaded");
  });

  it("does not throw when the version lookup errors", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <Harness />
      </QueryClientProvider>,
    );
    await screen.findByText("error");
  });

  it("does not fetch when disabled", () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <Harness enabled={false} />
      </QueryClientProvider>,
    );
    expect(mocks.get).not.toHaveBeenCalled();
  });
});

describe("shouldPromptReload", () => {
  it("is false before any version has loaded", () => {
    expect(shouldPromptReload(null, "v0.2.0")).toBe(false);
  });

  it("is false when the version has not changed", () => {
    expect(shouldPromptReload("v0.2.0", "v0.2.0")).toBe(false);
  });

  it("is true when the version changed since the tab first loaded", () => {
    expect(shouldPromptReload("v0.2.0", "v0.2.1")).toBe(true);
  });
});

describe("notifyIfServerUpdated", () => {
  const upgraded = { ...version, version: "v0.2.0-beta-331", update_available: false };

  beforeEach(() => {
    vi.resetModules();
  });

  it("goes to the network past a pre-upgrade cache and flags the new version as pending", async () => {
    const { notifyIfServerUpdated, useServerVersion: freshUseServerVersion, getServerVersionKey } =
      await import("@/hooks/VersionHooks");
    const { useServerUpdateStore } = await import("@/stores/serverUpdateStore");
    const client = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000 } } });
    const FreshHarness = () => {
      freshUseServerVersion();
      return null;
    };
    mocks.get.mockResolvedValueOnce({ data: version });
    render(
      <QueryClientProvider client={client}>
        <FreshHarness />
      </QueryClientProvider>,
    );
    await vi.waitFor(() => expect(client.getQueryData([getServerVersionKey])).toEqual(version));

    mocks.get.mockResolvedValueOnce({ data: upgraded });
    await notifyIfServerUpdated(client);

    expect(client.getQueryData([getServerVersionKey])).toEqual(upgraded);
    expect(useServerUpdateStore.getState().pendingVersion).toBe("v0.2.0-beta-331");
  });
});
