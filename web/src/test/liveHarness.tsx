import { act, render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { expect, vi } from "vitest";

import { api } from "@/api/client";
import type { LiveSocket } from "@/api/ws";
import { useLiveEvents } from "@/hooks/useLiveEvents";

// The calling test file mocks "@/api/client"; this harness drives that mock's get.
class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  close = vi.fn();
}

const requestOf = ([url, config]: unknown[]) => {
  const params = (config as { params?: Record<string, string> } | undefined)?.params;
  const query = params ? new URLSearchParams(params).toString() : "";
  return query ? `${url as string}?${query}` : (url as string);
};

// Mounts the live socket beside the given views, settles their first loads, then counts what each frame refetches.
export const mountLive = async (views: ReactNode, respond: (url: string) => unknown) => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => ({ data: respond(url) }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const sockets: FakeSocket[] = [];
  const wsFactory = () => {
    const s = new FakeSocket();
    sockets.push(s);
    return s;
  };
  const Live = () => {
    useLiveEvents("ws://live/ws/events", { wsFactory });
    return null;
  };
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Live />
        {views}
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await vi.waitFor(() => expect(sockets[0]?.onmessage).toBeTruthy());
  const socket = sockets[0]!;
  await vi.waitFor(() => expect(client.isFetching()).toBe(0));
  vi.mocked(api.get).mockClear();

  const requestsAfter = async (topic: string, payload: unknown) => {
    vi.mocked(api.get).mockClear();
    act(() => socket.onmessage?.({ data: JSON.stringify({ topic, type: "event", payload }) }));
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
    await vi.waitFor(() => expect(client.isFetching()).toBe(0));
    return vi.mocked(api.get).mock.calls.map(requestOf);
  };

  return { client, requestsAfter };
};
