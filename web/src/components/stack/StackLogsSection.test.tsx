import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useSyncExternalStore } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { StackLogsSection } from "@/components/stack/StackLogsSection";
import type { ContainerLogWireLine } from "@/models/ContainerLog";
import { useSessionStore } from "@/stores/sessionStore";

const access = vi.hoisted(() => ({ areas: ["stackLogs"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", async (importOriginal) => ({ ...(await importOriginal<object>()), api: { get: mocks.get } }));

// jsdom has no layout, so whether the reader has scrolled away from the newest line is set by hand.
const scroll = vi.hoisted(() => ({ away: false, listeners: new Set<() => void>() }));
vi.mock("@/components/ui/message-scroller", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useMessageScrollerScrollable: () => ({
    start: false,
    end: useSyncExternalStore(
      (notify) => {
        scroll.listeners.add(notify);
        return () => void scroll.listeners.delete(notify);
      },
      () => scroll.away,
    ),
  }),
}));

class FakeSocket {
  static all: FakeSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  readyState = FakeSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: { code: number; reason: string }) => void) | null = null;
  closed = false;
  constructor(readonly url: string) {
    FakeSocket.all.push(this);
  }
  close() {
    this.closed = true;
  }
}
const last = () => FakeSocket.all.at(-1)!;

const at = (n: number) => `2026-09-30T10:00:${String(n).padStart(2, "0")}.000000000Z`;
const line = (n: number, text = `line ${n}`, stream: ContainerLogWireLine["stream"] = "stdout"): ContainerLogWireLine => ({
  ts: at(n),
  stream,
  line: text,
});

const flush = () => act(() => vi.advanceTimersByTimeAsync(0));
const advance = (ms: number) => act(() => vi.advanceTimersByTimeAsync(ms));
const openSocket = () =>
  act(async () => {
    last().readyState = FakeSocket.OPEN;
    last().onopen?.();
  });
const emit = (lines: ContainerLogWireLine[]) => act(async () => last().onmessage?.({ data: JSON.stringify({ lines }) }));
const dropSocket = (code: number, reason = "") => act(async () => last().onclose?.({ code, reason }));
const scrollTo = (away: boolean) =>
  act(async () => {
    scroll.away = away;
    scroll.listeners.forEach((notify) => notify());
  });
const refusal = (status: number, message: string) => ({ response: { status, data: { message, code: "X" } } });

const services = [
  { id: "c-web", stack_id: "s-1", name: "web", declared: {}, status: "healthy" },
  { id: "c-db", stack_id: "s-1", name: "db", declared: {}, status: "healthy" },
];

const Section = () => <StackLogsSection stackId="s-1" service={useParams().service} />;

const renderSection = async (path = "/acme/stacks/s-1/logs/web") => {
  const view = render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/acme/stacks/:stackId/logs/:service?" element={<Section />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await flush();
  return view;
};

beforeEach(() => {
  vi.useFakeTimers();
  // Testing Library only drains fake timers around user events when it sees a jest-shaped global.
  vi.stubGlobal("jest", { advanceTimersByTime: vi.advanceTimersByTime });
  vi.stubGlobal("WebSocket", FakeSocket);
  FakeSocket.all = [];
  access.areas = ["stackLogs"];
  scroll.away = false;
  mocks.get.mockReset();
  mocks.get.mockImplementation((url: string) =>
    Promise.resolve({ data: url.endsWith("/services") ? services : { lines: [] } }),
  );
  useSessionStore.setState({ token: "tok", isLoggedIn: true });
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("StackLogsSection", () => {
  it("shows the refusal and opens nothing for a viewer without stacks:logs", async () => {
    access.areas = [];
    await renderSection();

    expect(screen.getByText("You can't read logs for this stack.")).toBeInTheDocument();
    expect(FakeSocket.all).toHaveLength(0);
    expect(mocks.get).not.toHaveBeenCalled();
  });

  it("renders lines in order over a socket carrying the token and the first tail", async () => {
    await renderSection();
    await openSocket();
    await emit([line(1), line(2, "connection refused", "stderr")]);

    expect(screen.getByText("line 1")).toBeInTheDocument();
    expect(screen.getByText("connection refused")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Live");
    expect(last().url).toBe("ws://localhost:8080/ws/stacks/s-1/services/web/logs?tail=200&token=tok");
  });

  it("says there is no output yet once the stream is live and empty", async () => {
    await renderSection();
    await openSocket();

    expect(screen.getAllByRole("status").map((el) => el.textContent)).toContain("No output yet");
  });

  it("Errors keeps lines that look like errors from either stream", async () => {
    const user = userEvent.setup({ delay: null });
    await renderSection();
    await openSocket();
    await emit([
      line(1, "listening on :8080"),
      line(2, "ERROR could not reach db"),
      line(3, "autovacuum launcher started", "stderr"),
      line(4, "goroutine 7 [running]:", "stderr"),
    ]);
    await user.click(screen.getByRole("radio", { name: "Errors" }));

    expect(screen.queryByText("listening on :8080")).not.toBeInTheDocument();
    expect(screen.queryByText("autovacuum launcher started")).not.toBeInTheDocument();
    expect(screen.getByText("ERROR could not reach db")).toBeInTheDocument();
    expect(screen.getByText("goroutine 7 [running]:")).toBeInTheDocument();
  });

  it("Errors says so when nothing looks like an error", async () => {
    const user = userEvent.setup({ delay: null });
    await renderSection();
    await openSocket();
    await emit([line(1)]);
    await user.click(screen.getByRole("radio", { name: "Errors" }));

    expect(screen.getAllByRole("status").map((el) => el.textContent)).toContain("No error output");
  });

  it("scrolling up says Scrolled up, and Pause says Paused and freezes the display until resumed", async () => {
    const user = userEvent.setup({ delay: null });
    await renderSection();
    await openSocket();
    await emit([line(1)]);

    await scrollTo(true);
    expect(screen.getByRole("status")).toHaveTextContent("Scrolled up");
    await scrollTo(false);

    await user.click(screen.getByRole("button", { name: "Pause" }));
    await emit([line(2)]);
    expect(screen.getByRole("status")).toHaveTextContent("Paused");
    expect(screen.queryByText("line 2")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Resume" }));
    expect(screen.getByText("line 2")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Live");
  });

  it("a dropped stream reconnects with a small tail and does not repeat lines it already has", async () => {
    await renderSection();
    await openSocket();
    await emit([line(1), line(2), line(3)]);
    await dropSocket(1006, "the runner disconnected");

    expect(screen.getByRole("status")).toHaveTextContent("Runner offline, reconnecting…");
    await advance(1_000);
    expect(FakeSocket.all).toHaveLength(2);
    expect(last().url).toContain("tail=20");

    await openSocket();
    await emit([line(2), line(3), line(4)]);
    expect(screen.getAllByText(/^line \d$/).map((n) => n.textContent)).toEqual(["line 1", "line 2", "line 3", "line 4"]);
  });

  it("a stream that ended normally stays ended", async () => {
    await renderSection();
    await openSocket();
    await emit([line(1)]);
    await dropSocket(1000, "the stream ended");

    expect(screen.getByRole("status")).toHaveTextContent("Stream ended");
    await advance(60_000);
    expect(FakeSocket.all).toHaveLength(1);
  });

  it("a refused handshake with a 403 explains itself and stops retrying", async () => {
    mocks.get.mockImplementation((url: string) =>
      url.endsWith("/services") ? Promise.resolve({ data: services }) : Promise.reject(refusal(403, "forbidden")),
    );
    await renderSection();
    await dropSocket(1006);

    expect(screen.getAllByRole("status").map((el) => el.textContent)).toContain("You can't read logs for this stack.");
    await advance(60_000);
    expect(FakeSocket.all).toHaveLength(1);
  });

  it("a refused handshake with the runner offline says so with the reason and retries", async () => {
    mocks.get.mockImplementation((url: string) =>
      url.endsWith("/services")
        ? Promise.resolve({ data: services })
        : Promise.reject(refusal(503, "no runner is connected on machine m1")),
    );
    await renderSection();
    await dropSocket(1006);

    expect(screen.getAllByRole("status").map((el) => el.textContent)).toContain(
      "Runner offline: no runner is connected on machine m1",
    );
    await advance(1_000);
    expect(FakeSocket.all).toHaveLength(2);
  });

  it("unmounting closes the socket", async () => {
    const { unmount } = await renderSection();
    await openSocket();
    const socket = last();
    unmount();

    expect(socket.closed).toBe(true);
  });

  it("unmounting cancels a pending reconnect", async () => {
    const { unmount } = await renderSection();
    await openSocket();
    await dropSocket(1006);
    unmount();

    await advance(60_000);
    expect(FakeSocket.all).toHaveLength(1);
  });

  it("Copy puts the lines shown, and only those, on the clipboard", async () => {
    const user = userEvent.setup({ delay: null });
    await renderSection();
    await openSocket();
    await emit([line(1, "listening on :8080"), line(2, "ERROR could not reach db")]);
    await user.click(screen.getByRole("radio", { name: "Errors" }));
    const write = vi.spyOn(navigator.clipboard, "writeText");
    await user.click(screen.getByRole("button", { name: "Copy" }));

    expect(write).toHaveBeenCalledOnce();
    const copied = String(write.mock.calls[0]?.[0]);
    expect(copied).toContain("ERROR could not reach db");
    expect(copied).not.toContain("listening");
  });

  it("switching tabs closes the first socket and opens the next service's", async () => {
    const user = userEvent.setup({ delay: null });
    await renderSection();
    await openSocket();
    const web = last();
    await emit([line(1, "web line")]);

    await user.click(screen.getByRole("tab", { name: "db" }));
    await flush();

    expect(web.closed).toBe(true);
    expect(FakeSocket.all).toHaveLength(2);
    expect(last().url).toContain("/services/db/logs");
    expect(screen.queryByText("web line")).not.toBeInTheDocument();
  });
});
