import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { useReportTimezone } from "@/hooks/AuthHooks";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), put: jest.fn() } }));

notifyManager.setScheduler((callback) => callback());

const device = () => Intl.DateTimeFormat().resolvedOptions().timeZone;

const mount = async (signedIn = true) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  const view = await renderHook(() => useReportTimezone(signedIn), { wrapper });
  return { client, ...view };
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  jest.mocked(api.put).mockReset().mockResolvedValue(undefined);
});

describe("useReportTimezone", () => {
  test("reports the phone's zone once when the server has none", async () => {
    jest.mocked(api.get).mockResolvedValue({ user: { id: "u1", login: "onik97", name: "Onik" } });
    const { rerender } = await mount();

    await waitFor(() => expect(api.put).toHaveBeenCalledWith("/api/auth/timezone", { timezone: device() }));
    await rerender({});
    expect(api.put).toHaveBeenCalledTimes(1);
  });

  test("reports a zone that differs from the stored one", async () => {
    jest.mocked(api.get).mockResolvedValue({ user: { id: "u1", login: "onik97", name: "Onik", timezone: "Pacific/Chatham" } });
    await mount();

    await waitFor(() => expect(api.put).toHaveBeenCalledWith("/api/auth/timezone", { timezone: device() }));
  });

  test("stays quiet when the stored zone is already the phone's", async () => {
    jest.mocked(api.get).mockResolvedValue({ user: { id: "u1", login: "onik97", name: "Onik", timezone: device() } });
    const { client } = await mount();
    await waitFor(() => expect(client.getQueryState(["getMe"])?.status).toBe("success"));

    expect(api.put).not.toHaveBeenCalled();
  });

  test("sends nothing before anyone is signed in", async () => {
    await mount(false);

    expect(api.get).not.toHaveBeenCalled();
    expect(api.put).not.toHaveBeenCalled();
  });
});
