import { act, render, screen, userEvent } from "@testing-library/react-native";
import { AppState, type AppStateStatus } from "react-native";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { ContainerLogsScreen } from "@/components/deploys/ContainerLogsScreen";
import { MAX_LOG_LINES } from "@/hooks/ContainerLogHooks";
import type { ContainerLogLine } from "@/models/Stack";
import { useSessionStore } from "@/stores/sessionStore";

jest.mock("expo-secure-store", () => {
  const mockSecrets = new Map<string, string>();
  return {
    getItem: (key: string) => mockSecrets.get(key) ?? null,
    setItem: (key: string, value: string) => void mockSecrets.set(key, value),
    deleteItemAsync: async (key: string) => void mockSecrets.delete(key),
  };
});

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const mockAccess: { current: boolean | undefined } = { current: true };
jest.mock("@/hooks/WorkspaceHooks", () => ({ useAreaAccess: () => (mockAccess.current === undefined ? undefined : () => mockAccess.current) }));

// The screen's own focus lifecycle: a test blurs and refocuses it the way the navigator would.
const mockFocus = new Set<{ run: () => void; stop: () => void }>();
jest.mock("expo-router", () => {
  const { useEffect } = jest.requireActual<typeof import("react")>("react");
  return {
    Stack: { Screen: () => null },
    useLocalSearchParams: () => ({ id: "s-1", service: "web" }),
    useFocusEffect: (effect: () => void | (() => void)) =>
      useEffect(() => {
        let cleanup: void | (() => void);
        const handle = {
          run: () => {
            cleanup = effect();
          },
          stop: () => {
            if (typeof cleanup === "function") cleanup();
            cleanup = undefined;
          },
        };
        handle.run();
        mockFocus.add(handle);
        return () => {
          handle.stop();
          mockFocus.delete(handle);
        };
      }, [effect]),
  };
});

// Jest has no layout pass, so this stand-in renders every row in data order and exposes the scroll props and ref.
const mockList: { props: Record<string, unknown>; scrollToEnd: jest.Mock } = { props: {}, scrollToEnd: jest.fn() };
jest.mock("@legendapp/list/react-native", () => {
  const { Fragment, createElement, forwardRef, useImperativeHandle } = jest.requireActual<typeof import("react")>("react");
  return {
    LegendList: forwardRef(
      (
        props: { data: unknown[]; keyExtractor: (item: unknown) => string; renderItem: (info: { item: unknown }) => unknown },
        ref: unknown,
      ) => {
        useImperativeHandle(ref as never, () => ({ scrollToEnd: mockList.scrollToEnd }));
        mockList.props = props;
        return props.data.map((item) => createElement(Fragment, { key: props.keyExtractor(item) }, props.renderItem({ item }) as never));
      },
    ),
  };
});

class FakeSocket {
  static all: FakeSocket[] = [];
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
const line = (n: number, text = `line ${n}`, stream: ContainerLogLine["stream"] = "stdout"): ContainerLogLine => ({ ts: at(n), stream, line: text });

const openSocket = () => act(async () => last().onopen?.());
const emit = (lines: ContainerLogLine[]) => act(async () => last().onmessage?.({ data: JSON.stringify({ lines }) }));
const dropSocket = (code: number, reason = "") => act(async () => last().onclose?.({ code, reason }));
const scroll = (distanceFromEnd: number) =>
  act(async () =>
    (mockList.props.onScroll as (e: unknown) => void)({
      nativeEvent: { contentOffset: { y: 1000 - distanceFromEnd }, contentSize: { height: 1600 }, layoutMeasurement: { height: 600 } },
    }),
  );

let appStateChange: (status: AppStateStatus) => void = () => undefined;

beforeEach(() => {
  jest.useFakeTimers();
  Object.assign(AppState, { currentState: "active" });
  jest.spyOn(AppState, "addEventListener").mockImplementation((_, listener) => {
    appStateChange = listener as (status: AppStateStatus) => void;
    return { remove: jest.fn() };
  });
  FakeSocket.all = [];
  mockFocus.clear();
  mockAccess.current = true;
  mockList.scrollToEnd.mockReset();
  jest.mocked(api.get).mockReset();
  global.WebSocket = FakeSocket as unknown as typeof WebSocket;
  useSessionStore.getState().signIn("https://nexul.example.com", "tok");
});

afterEach(() => {
  jest.useRealTimers();
  jest.restoreAllMocks();
});

const advance = (ms: number) => act(async () => void jest.advanceTimersByTime(ms));

describe("ContainerLogsScreen", () => {
  test("a viewer without stacks:logs sees the refusal and never opens a socket", async () => {
    mockAccess.current = false;
    await render(<ContainerLogsScreen />);

    expect(screen.getByText("No access to these logs")).toBeTruthy();
    expect(FakeSocket.all).toHaveLength(0);
  });

  test("a refused handshake with a 403 explains itself and stops retrying", async () => {
    jest.mocked(api.get).mockRejectedValue(new ApiError(403, { message: "forbidden", code: "FORBIDDEN" }, "GET failed"));
    await render(<ContainerLogsScreen />);
    await dropSocket(1006);

    expect(await screen.findByText("No access to these logs")).toBeTruthy();
    await advance(60_000);
    expect(FakeSocket.all).toHaveLength(1);
  });

  test("a refused handshake with the runner offline says so and retries", async () => {
    jest.mocked(api.get).mockRejectedValue(new ApiError(503, { message: "no runner is connected on machine m1", code: "RETRYABLE" }, "GET failed"));
    await render(<ContainerLogsScreen />);
    await dropSocket(1006);

    expect(await screen.findByText("Runner offline")).toBeTruthy();
    expect(screen.getByText("no runner is connected on machine m1")).toBeTruthy();
    await advance(1_000);
    expect(FakeSocket.all).toHaveLength(2);
  });

  test("no output yet once the stream is live and empty", async () => {
    await render(<ContainerLogsScreen />);
    await openSocket();

    expect(screen.getByText("No output yet")).toBeTruthy();
  });

  test("lines render in order, and the socket carries the session token and the first tail", async () => {
    await render(<ContainerLogsScreen />);
    await openSocket();
    await emit([line(1), line(2, "boom", "stderr")]);

    expect(screen.getByText("line 1")).toBeTruthy();
    expect(screen.getByText("boom")).toBeTruthy();
    expect(screen.getByText("Live")).toBeTruthy();
    expect(last().url).toBe("wss://nexul.example.com/ws/stacks/s-1/services/web/logs?tail=200&token=tok");
  });

  test("a dropped stream reconnects with a small tail and does not repeat lines it already has", async () => {
    await render(<ContainerLogsScreen />);
    await openSocket();
    await emit([line(1), line(2), line(3)]);
    await dropSocket(1006, "the runner disconnected");

    expect(screen.getByText("Runner offline, reconnecting…")).toBeTruthy();
    await advance(1_000);
    expect(FakeSocket.all).toHaveLength(2);
    expect(last().url).toContain("tail=20");

    await openSocket();
    await emit([line(2), line(3), line(4)]);
    expect(screen.getAllByText(/^line \d$/).map((n) => n.props.children)).toEqual(["line 1", "line 2", "line 3", "line 4"]);
  });

  test("a stream that ended normally stays ended", async () => {
    await render(<ContainerLogsScreen />);
    await openSocket();
    await emit([line(1)]);
    await dropSocket(1000, "the stream ended");

    expect(screen.getByText("Stream ended")).toBeTruthy();
    await advance(60_000);
    expect(FakeSocket.all).toHaveLength(1);
  });

  test("scrolling up pauses following and Jump to live resumes it", async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime });
    await render(<ContainerLogsScreen />);
    await openSocket();
    await emit([line(1)]);
    expect(screen.queryByText("Jump to live")).toBeNull();

    await scroll(400);
    expect(screen.getByText("Paused")).toBeTruthy();
    await user.press(screen.getByRole("button", { name: "Jump to live" }));

    expect(mockList.scrollToEnd).toHaveBeenCalled();
    expect(screen.queryByText("Jump to live")).toBeNull();
    expect(screen.getByText("Live")).toBeTruthy();
  });

  test("blur closes the socket, so the host stops streaming, and refocus starts a fresh one", async () => {
    await render(<ContainerLogsScreen />);
    await openSocket();
    const first = last();

    await act(async () => mockFocus.forEach((h) => h.stop()));
    expect(first.closed).toBe(true);
    await advance(60_000);
    expect(FakeSocket.all).toHaveLength(1);

    await act(async () => mockFocus.forEach((h) => h.run()));
    expect(FakeSocket.all).toHaveLength(2);
  });

  test("backgrounding the app closes the socket, and returning reopens it with the small tail, keeping the lines", async () => {
    await render(<ContainerLogsScreen />);
    await openSocket();
    await emit([line(1), line(2)]);
    const first = last();

    await act(async () => appStateChange("background"));
    expect(first.closed).toBe(true);
    await advance(60_000);
    expect(FakeSocket.all).toHaveLength(1);

    await act(async () => appStateChange("active"));
    expect(FakeSocket.all).toHaveLength(2);
    expect(last().url).toContain("tail=20");
    await openSocket();
    await emit([line(2), line(3)]);
    expect(screen.getAllByText(/^line \d$/).map((n) => n.props.children)).toEqual(["line 1", "line 2", "line 3"]);
  });

  test("only the last 2000 lines are kept", async () => {
    await render(<ContainerLogsScreen />);
    await openSocket();
    const many = Array.from({ length: MAX_LOG_LINES + 50 }, (_, i) => ({ ts: `2026-09-30T11:00:00.${String(i).padStart(9, "0")}Z`, stream: "stdout" as const, line: `n${i}` }));
    await emit(many);

    const data = mockList.props.data as { line: string }[];
    expect(data).toHaveLength(MAX_LOG_LINES);
    expect(data[0]?.line).toBe("n50");
    expect(data.at(-1)?.line).toBe(`n${MAX_LOG_LINES + 49}`);
  });
});
