import type { Terminal } from "@xterm/xterm";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { connectTerminal, type TerminalState } from "@/lib/terminal";

class FakeSocket {
  static OPEN = 1;
  static last: FakeSocket;
  readyState = 0;
  binaryType = "blob";
  sent: (string | Uint8Array)[] = [];
  closed = false;
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: ArrayBuffer | string }) => void) | null = null;
  onclose: (() => void) | null = null;
  constructor(readonly url: string) {
    FakeSocket.last = this;
  }
  send(data: string | Uint8Array) {
    this.sent.push(data);
  }
  close() {
    this.closed = true;
  }
  open() {
    this.readyState = FakeSocket.OPEN;
    this.onopen?.();
  }
}

// The parts of an xterm the socket drives, with its listeners kept so the test can type and resize.
const fakeTerm = () => {
  const listeners: Record<string, (arg: never) => void> = {};
  const on = (name: string) => (fn: (arg: never) => void) => {
    listeners[name] = fn;
    return { dispose: () => delete listeners[name] };
  };
  const term = {
    cols: 80,
    rows: 24,
    written: [] as string[],
    focus: vi.fn(),
    options: {} as Record<string, unknown>,
    onData: on("data"),
    onBinary: on("binary"),
    onResize: on("resize"),
  };
  const write = (data: Uint8Array | string) => term.written.push(typeof data === "string" ? data : new TextDecoder().decode(data));
  const fire = (name: string, arg: unknown) => listeners[name]?.(arg as never);
  return {
    term: { ...term, write } as unknown as Terminal,
    listeners,
    fire,
    written: term.written,
  };
};

describe("connectTerminal", () => {
  beforeEach(() => vi.stubGlobal("WebSocket", FakeSocket));
  afterEach(() => vi.unstubAllGlobals());

  it("sends its size on open, keys as binary and a resize as text, and writes what the shell prints", () => {
    const { term, fire, written } = fakeTerm();
    const states: TerminalState[] = [];
    connectTerminal(term, "ws://nexul/ws/pairing/terminals/t1?token=x", (s) => states.push(s));
    const ws = FakeSocket.last;
    expect(ws.binaryType).toBe("arraybuffer");
    fire("data", "dropped before open");
    ws.open();

    expect(states).toEqual([{ status: "live" }]);
    expect(ws.sent).toEqual(['{"cols":80,"rows":24}']);
    fire("data", "ls\r");
    expect(new TextDecoder().decode(ws.sent[1] as Uint8Array)).toBe("ls\r");
    (term as { cols: number }).cols = 120;
    fire("resize", {});
    expect(ws.sent[2]).toBe('{"cols":120,"rows":24}');
    ws.onmessage?.({ data: new TextEncoder().encode("Desktop\r\n").buffer });
    expect(written).toEqual(["Desktop\r\n"]);
  });

  it("ends with the shell's exit, or as lost when the socket closes without one", () => {
    const { term } = fakeTerm();
    const states: TerminalState[] = [];
    connectTerminal(term, "ws://x", (s) => states.push(s));
    FakeSocket.last.open();
    FakeSocket.last.onmessage?.({ data: '{"exit":3}' });
    FakeSocket.last.onclose?.();
    expect(term.options.disableStdin).toBe(true);
    connectTerminal(term, "ws://y", (s) => states.push(s));
    FakeSocket.last.onclose?.();
    expect(states).toEqual([{ status: "live" }, { status: "ended", exit: 3 }, { status: "lost" }]);
  });

  it("closes the socket, which ends the shell, when the screen goes away", () => {
    const { term, listeners } = fakeTerm();
    const states: TerminalState[] = [];
    const disconnect = connectTerminal(term, "ws://x", (s) => states.push(s));
    disconnect();
    expect(FakeSocket.last.closed).toBe(true);
    expect(FakeSocket.last.onclose).toBeNull();
    expect(listeners).toEqual({});
  });
});
