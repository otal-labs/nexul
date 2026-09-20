import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import * as Y from "yjs";

import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { RelayCollabProvider } from "@/lib/collab/provider";
import type { LiveSocket } from "@/api/ws";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  send = vi.fn();
  close = vi.fn();

  dispatch(raw: string) {
    this.onmessage?.({ data: raw });
  }
}

const structured = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`;

const connect = (socket: FakeSocket) => {
  socket.onopen?.({});
};

const wrap = (props: Parameters<typeof RichTextEditor>[0]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <RichTextEditor {...props} />
      </MemoryRouter>
    </QueryClientProvider>
  );
};

const renderEditor = (props: Parameters<typeof RichTextEditor>[0]) => render(wrap(props));

describe("RichTextEditor collab mode", () => {
  it("binds a session and seeds content only when the server has none", async () => {
    const socket = new FakeSocket();
    const doc = new Y.Doc();
    const provider = new RelayCollabProvider({
      url: "ws://test",
      doc,
      wsFactory: () => socket,
    });
    const { rerender } = renderEditor({
      value: structured,
      onChange: () => {},
      collab: { doc, provider, user: { name: "Alice", color: "#3b82f6" }, serverReady: false, hasServerState: false },
    });
    expect(screen.getByLabelText(/doc body/i)).toBeInTheDocument();

    // Server confirms an empty store, so the editor seeds the body into the session
    // (an update frame leaves for the server).
    provider.connect();
    connect(socket);
    rerender(
      wrap({
        value: structured,
        onChange: () => {},
        collab: { doc, provider, user: { name: "Alice", color: "#3b82f6" }, serverReady: true, hasServerState: false },
      }),
    );
    await waitFor(() => {
      const updates = socket.send.mock.calls
        .map((c) => JSON.parse(c[0] as string) as { type: string })
        .filter((f) => f.type === "update");
      expect(updates.length).toBeGreaterThan(0);
    });
    provider.destroy();
  });

  it("never seeds when the server replays stored state", async () => {
    const socket = new FakeSocket();
    const doc = new Y.Doc();
    const provider = new RelayCollabProvider({
      url: "ws://test",
      doc,
      wsFactory: () => socket,
    });
    provider.connect();
    connect(socket);
    renderEditor({
      value: structured,
      onChange: () => {},
      collab: { doc, provider, user: { name: "Alice", color: "#3b82f6" }, serverReady: true, hasServerState: true },
    });
    const updates = socket.send.mock.calls
      .map((c) => JSON.parse(c[0] as string) as { type: string })
      .filter((f) => f.type === "update");
    expect(updates).toHaveLength(0);
    provider.destroy();
  });
});
