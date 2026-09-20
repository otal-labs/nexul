import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocDetail } from "@/components/doc/DocDetail";
import { getMeKey } from "@/hooks/AuthHooks";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { LiveSocket } from "@/api/ws";
import type { Doc } from "@/models/Doc";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  resolveWSBase: vi.fn(() => "ws://test"),
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

const doc: Doc = {
  id: "doc-1",
  project_id: "p-1",
  title: "Storage Spine",
  body: "SQLite is the spine.",
  version: 2,
  archived: false,
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const framesOf = (socket: FakeSocket, type: string) =>
  socket.send.mock.calls
    .map((c) => JSON.parse(c[0] as string) as { type: string })
    .filter((f) => f.type === type);

// connect() defers by a tick (ws-25, see useCollabSession.ts) so StrictMode's
// throwaway instance never opens a real socket; tests must use flushConnect()
// (or advance fake timers) before FakeSocket's handlers apply.
const flushConnect = () => act(() => new Promise((r) => setTimeout(r, 0)));

const renderDetailRaw = (socket: FakeSocket) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  // Prime the profile cache (production warms it via OnboardingGate): the collab
  // session's `name` is a session-rebuilding dep, so it must be present on the first render.
  client.setQueryData([getMeKey], {
    user: { name: "Alice" },
    needs_owner_wizard: false,
    needs_first_login_wizard: false,
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DocDetail
          doc={doc}
          workspaceId="ws-1"
          wsFactory={() => socket}
          onCreateTicket={vi.fn()}
          onPermissions={vi.fn()}
          onArchive={vi.fn()}
          onRestore={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const renderDetail = async (socket: FakeSocket) => {
  const result = renderDetailRaw(socket);
  await flushConnect();
  return result;
};

beforeEach(() => {
  useSessionStore.setState({ token: "token-1", isLoggedIn: true });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  // The collab display name comes from GET /api/auth/me (useFetchMe); PlaysMenu's
  // harness-readiness check needs a well-shaped presence map; every other GET
  // here (tickets/mentions) stays empty.
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: { user: { name: "Alice" }, needs_owner_wizard: false, needs_first_login_wizard: false } };
    if (url === "/api/pairing/presence") return { data: { computers: {} } };
    if (url === "/api/pairing/resolve") return { data: { ok: false, reason: "unpaired" } };
    return { data: [] };
  });
  vi.mocked(api.post).mockResolvedValue({ data: { chips: [] } });
});

describe("DocDetail", () => {
  it("is live by default: no edit toggle, the page is the editor", async () => {
    const socket = new FakeSocket();
    await renderDetail(socket);

    expect(screen.getByTestId("doc-title-input")).toHaveValue("Storage Spine");
    expect(await screen.findByLabelText("Body")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
  });

  it("joins the live edit session and announces presence", async () => {
    const socket = new FakeSocket();
    await renderDetail(socket);
    socket.onopen?.({});

    expect(framesOf(socket, "hello")).toHaveLength(1);
    expect(framesOf(socket, "presence").length).toBeGreaterThan(0);
  });

  it("seeds a fresh doc body once the server confirms it has no state", async () => {
    const socket = new FakeSocket();
    await renderDetail(socket);
    socket.onopen?.({});
    socket.dispatch(
      JSON.stringify({ type: "init", seq: 0, snapshot: null, updates: [], presence: [] }),
    );

    await waitFor(() => expect(framesOf(socket, "update").length).toBeGreaterThan(0));
  });

  it("does not seed when the server replays stored state", async () => {
    const socket = new FakeSocket();
    await renderDetail(socket);
    socket.onopen?.({});
    socket.dispatch(
      JSON.stringify({
        type: "init",
        seq: 4,
        snapshot: { seq: 4, kind: "snapshot", actor_id: "bob", payload: "bm8tYm9keQ==" },
        updates: [],
        presence: [],
      }),
    );

    await waitFor(() => expect(framesOf(socket, "update")).toHaveLength(0));
  });

  it("commits the converged title and body on the commit cadence", async () => {
    vi.useFakeTimers();
    try {
      const socket = new FakeSocket();
      renderDetailRaw(socket);
      // Fake timers own setTimeout here, so flush the deferred connect() by advancing
      // the clock instead of flushConnect()'s real-timer wait.
      await act(async () => {
        vi.advanceTimersByTime(0);
      });
      socket.onopen?.({});
      await act(async () => {
        socket.dispatch(
          JSON.stringify({ type: "init", seq: 0, snapshot: null, updates: [], presence: [] }),
        );
      });
      // Let the seed + body onChange settle, then cross one commit interval.
      await act(async () => {
        vi.advanceTimersByTime(6000);
      });

      const commits = framesOf(socket, "commit") as unknown as { title?: string; body: string }[];
      expect(commits.length).toBeGreaterThan(0);
      // An untouched title is omitted from commits ("" → unchanged), so this
      // client's mount-time title can never clobber a peer's rename.
      expect(commits[0]?.title ?? "").toBe("");
      expect(commits[0]?.body).toContain("doc");
    } finally {
      vi.useRealTimers();
    }
  });

  it("edits the title inline", async () => {
    const user = userEvent.setup();
    const socket = new FakeSocket();
    await renderDetail(socket);
    socket.onopen?.({});

    const title = screen.getByTestId("doc-title-input");
    await user.clear(title);
    await user.type(title, "Storage Spine v2");

    expect(title).toHaveValue("Storage Spine v2");
  });

  it("Enter commits the rename immediately, carrying the typed title", async () => {
    const user = userEvent.setup();
    const socket = new FakeSocket();
    await renderDetail(socket);
    socket.onopen?.({});
    await act(async () => {
      socket.dispatch(JSON.stringify({ type: "init", seq: 0, snapshot: null, updates: [], presence: [] }));
    });

    const title = screen.getByTestId("doc-title-input");
    await user.clear(title);
    await user.type(title, "Renamed by me{Enter}");

    const commits = framesOf(socket, "commit") as unknown as { title?: string }[];
    expect(commits.length).toBeGreaterThan(0);
    expect(commits.at(-1)?.title).toBe("Renamed by me");
  });

  it("applies a peer's rename from the commit relay", async () => {
    const socket = new FakeSocket();
    await renderDetail(socket);
    socket.onopen?.({});
    await act(async () => {
      socket.dispatch(JSON.stringify({ type: "init", seq: 0, snapshot: null, updates: [], presence: [] }));
      socket.dispatch(JSON.stringify({ type: "commit", from: "bob", seq: 2, title: "Renamed by peer" }));
    });

    expect(screen.getByTestId("doc-title-input")).toHaveValue("Renamed by peer");
  });

  it("shows presence avatars for remote participants", async () => {
    const socket = new FakeSocket();
    await renderDetail(socket);
    socket.onopen?.({});

    const { Awareness, encodeAwarenessUpdate } = await import("y-protocols/awareness");
    const { Doc } = await import("yjs");
    const remote = new Awareness(new Doc());
    remote.clientID = 42;
    // Set twice: y-protocols skips a client's very first state (clock 0).
    remote.setLocalState({ user: { name: "Bob", color: "#3b82f6", activity: "editing" } });
    remote.setLocalState({ user: { name: "Bob", color: "#3b82f6", activity: "editing" } });
    socket.dispatch(
      JSON.stringify({
        type: "presence",
        from: "bob",
        client_id: 42,
        payload: btoa(String.fromCharCode(...encodeAwarenessUpdate(remote, [42]))),
      }),
    );

    expect(await screen.findByLabelText("Bob editing")).toBeInTheDocument();
  });

  it("offers the conflict fallback on an unresolvable update", async () => {
    const user = userEvent.setup();
    const socket = new FakeSocket();
    await renderDetail(socket);
    socket.onopen?.({});

    socket.dispatch(
      JSON.stringify({ type: "update", from: "bob", seq: 1, payload: btoa("\u0009\u0009\u0009") }),
    );

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Use server version" }));
    // The reset remounts a fresh session over the same test socket; the new provider
    // connects once its deferred connect()'s tick passes, so opening the socket again
    // announces a brand-new Y client.
    await flushConnect();
    socket.onopen?.({});
    const hellos = framesOf(socket, "hello");
    expect(hellos.length).toBeGreaterThanOrEqual(2);
    const ids = new Set(hellos.map((h) => (h as unknown as { client_id: number }).client_id));
    expect(ids.size).toBe(2);
  });

  it("shows the doc's Trail section for a caller with docs:thread", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: { name: "Alice" }, needs_owner_wizard: false, needs_first_login_wizard: false } };
      if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["docs:thread"] } };
      if (url === "/api/pairing/presence") return { data: { computers: {} } };
      if (url === "/api/pairing/resolve") return { data: { ok: false, reason: "unpaired" } };
      if (url === "/api/plays/runs") {
        return {
          data: [
            {
              id: "tr-1",
              workspace_id: "ws-1",
              play_id: "play-doc",
              play_label: "To tickets via AI",
              target_type: "doc",
              target_id: "doc-1",
              project_id: "p-1",
              conversation_id: "c-1",
              starter_id: "u-1",
              via: "web",
              selected_memory_ids: [],
              custom_instructions: "",
              move_to_status_id: "",
              computer_id: "",
              provider: "",
              model: "",
              harness_session_id: "",
              state: "done",
              started_at: "2026-09-17T10:00:00Z",
              ended_at: "2026-09-17T10:01:00Z",
              last_error: "",
              reply_message_id: "",
              activity: [],
            },
          ],
        };
      }
      return { data: [] };
    });
    const socket = new FakeSocket();
    await renderDetail(socket);

    expect(await screen.findByRole("heading", { name: "Trail" })).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith("/api/plays/runs", { params: { target_type: "doc", target_id: "doc-1" } });
  });

  it("hides the Trail section for a caller without docs:thread", async () => {
    const socket = new FakeSocket();
    await renderDetail(socket);

    expect(screen.queryByRole("heading", { name: "Trail" })).not.toBeInTheDocument();
  });
});
