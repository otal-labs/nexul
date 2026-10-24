import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useReportTimezone } from "@/hooks/useReportTimezone";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(), put: vi.fn() }, errorMessage: vi.fn() }));

const device = Intl.DateTimeFormat().resolvedOptions().timeZone;

const meWith = (timezone?: string) => ({ data: { user: { id: "u-1", login: "alice", ...(timezone && { timezone }) } } });

const render = (signedIn = true) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  return renderHook(({ on }) => useReportTimezone(on), { wrapper, initialProps: { on: signedIn } });
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset().mockResolvedValue({ data: undefined });
});

describe("useReportTimezone", () => {
  it("reports the browser's zone once when the server holds another", async () => {
    vi.mocked(api.get).mockResolvedValue(meWith("Pacific/Kiritimati"));
    const view = render();
    await waitFor(() => expect(api.put).toHaveBeenCalledWith("/api/auth/timezone", { timezone: device }));
    view.rerender({ on: true });
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2));
    expect(api.put).toHaveBeenCalledTimes(1);
  });

  it("reports when the server holds none", async () => {
    vi.mocked(api.get).mockResolvedValue(meWith());
    render();
    await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1));
  });

  it("stays quiet when the server already holds the browser's zone", async () => {
    vi.mocked(api.get).mockResolvedValue(meWith(device));
    render();
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(1));
    expect(api.put).not.toHaveBeenCalled();
  });

  it("asks nothing while signed out", () => {
    render(false);
    expect(api.get).not.toHaveBeenCalled();
    expect(api.put).not.toHaveBeenCalled();
  });
});
