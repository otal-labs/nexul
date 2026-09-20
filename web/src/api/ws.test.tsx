import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { LiveEventsClient, parseFrame, type LiveSocket } from "@/api/ws";

class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  close = vi.fn();

  open() {
    this.onopen?.(null);
  }

  message(data: string) {
    this.onmessage?.({ data });
  }

  closeFromServer() {
    this.onclose?.(null);
  }
}

describe("parseFrame", () => {
  it("parses a well-formed frame", () => {
    expect(parseFrame('{"topic":"topology","type":"updated","payload":{}}')).toEqual({
      topic: "topology",
      type: "updated",
      payload: {},
    });
  });

  it("throws on malformed JSON", () => {
    expect(() => parseFrame("not-json")).toThrow();
  });

  it("throws when topic is missing", () => {
    expect(() => parseFrame('{"type":"updated"}')).toThrow();
  });

  it("throws when payload is not an object", () => {
    expect(() => parseFrame('"just-a-string"')).toThrow();
  });
});

describe("LiveEventsClient", () => {
  let sockets: FakeSocket[];
  let client: LiveEventsClient;

  const factory = (): LiveSocket => {
    const socket = new FakeSocket();
    sockets.push(socket);
    return socket;
  };

  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(Math, "random").mockReturnValue(0);
    sockets = [];
    client = new LiveEventsClient("ws://test", { wsFactory: factory });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("opens a socket and resets the attempt counter on open", () => {
    const handler = vi.fn();
    client.subscribe(handler);
    client.connect();
    expect(sockets).toHaveLength(1);
    sockets[0]?.open();
    expect(handler).not.toHaveBeenCalled();
  });

  it("dispatches parsed frames to subscribers", () => {
    const handler = vi.fn();
    client.subscribe(handler);
    client.connect();
    sockets[0]?.open();
    sockets[0]?.message('{"topic":"tickets","type":"updated","payload":{"id":"t1"}}');
    expect(handler).toHaveBeenCalledWith({
      topic: "tickets",
      type: "updated",
      payload: { id: "t1" },
    });
  });

  it("drops malformed frames without dispatching", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const handler = vi.fn();
    client.subscribe(handler);
    client.connect();
    sockets[0]?.message("garbage");
    expect(handler).not.toHaveBeenCalled();
    expect(warn).toHaveBeenCalled();
  });

  it("unsubscribes a handler", () => {
    const handler = vi.fn();
    const unsubscribe = client.subscribe(handler);
    unsubscribe();
    client.connect();
    sockets[0]?.message('{"topic":"x","type":"y","payload":{}}');
    expect(handler).not.toHaveBeenCalled();
  });

  it("reconnects with exponential backoff after close", () => {
    client.connect();
    sockets[0]?.closeFromServer();
    vi.advanceTimersByTime(500);
    expect(sockets).toHaveLength(2);
  });

  it("grows the backoff delay between attempts", () => {
    client.connect();
    sockets[0]?.closeFromServer();
    vi.advanceTimersByTime(500);
    expect(sockets).toHaveLength(2);
    sockets[1]?.closeFromServer();
    vi.advanceTimersByTime(500);
    expect(sockets).toHaveLength(2);
    vi.advanceTimersByTime(500);
    expect(sockets).toHaveLength(3);
  });

  it("does not reconnect after close()", () => {
    client.connect();
    client.close();
    sockets[0]?.closeFromServer();
    vi.advanceTimersByTime(60_000);
    expect(sockets).toHaveLength(1);
  });

  it("fires onReconnect on every open after the first, not on the initial connect", () => {
    const onReconnect = vi.fn();
    const reconnecting = new LiveEventsClient("ws://test", { wsFactory: factory, onReconnect });
    reconnecting.connect();
    sockets[0]?.open();
    expect(onReconnect).not.toHaveBeenCalled();

    sockets[0]?.closeFromServer();
    vi.advanceTimersByTime(500);
    sockets[1]?.open();
    expect(onReconnect).toHaveBeenCalledTimes(1);

    sockets[1]?.closeFromServer();
    vi.advanceTimersByTime(500);
    sockets[2]?.open();
    expect(onReconnect).toHaveBeenCalledTimes(2);
  });
});
