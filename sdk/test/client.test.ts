import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { defineAutomation } from "../src/define-automation.ts";
import { DialinClient } from "../src/client.ts";
import type { ConfigSchema } from "../src/config-schema.ts";

// FakeWebSocket stands in for the real WebSocket so these tests exercise the
// wire protocol (announce -> hello -> event -> run report) without a real
// server or network — the same approach the mock-first SDK asks its own
// users to take.
class FakeWebSocket extends EventTarget {
  static instances: FakeWebSocket[] = [];
  sent: unknown[] = [];
  readyState = 0;

  constructor(public url: string) {
    super();
    FakeWebSocket.instances.push(this);
    queueMicrotask(() => {
      this.readyState = 1;
      this.dispatchEvent(new Event("open"));
    });
  }

  send(data: string): void {
    this.sent.push(JSON.parse(data));
  }

  close(): void {
    this.readyState = 3;
    this.dispatchEvent(new Event("close"));
  }

  emit(frame: unknown): void {
    this.dispatchEvent(Object.assign(new Event("message"), { data: JSON.stringify(frame) }));
  }
}

function tick(ms = 5): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

let realWebSocket: typeof WebSocket;

beforeEach(() => {
  realWebSocket = globalThis.WebSocket;
  FakeWebSocket.instances = [];
  globalThis.WebSocket = FakeWebSocket as unknown as typeof WebSocket;
});

afterEach(() => {
  globalThis.WebSocket = realWebSocket;
});

async function connect<S extends ConfigSchema>(client: DialinClient<S>) {
  const runPromise = client.run();
  await tick();
  const ws = FakeWebSocket.instances[0]!;
  ws.emit({ type: "hello", config_values: {}, secrets: { API_KEY: "s3cr3t" } });
  await tick();
  return { ws, runPromise };
}

describe("DialinClient protocol", () => {
  it("announces name, description and subscriptions on connect", async () => {
    const automation = defineAutomation({ name: "board-pair", description: "moves tickets" });
    automation.on("ticket.created", async () => true);
    const client = new DialinClient(automation, { url: "http://instance", token: "dat_x" });

    const { ws } = await connect(client);

    expect(ws.sent[0]).toMatchObject({ type: "announce", name: "board-pair", description: "moves tickets", subscriptions: ["ticket.created"] });
    client.stop();
  });

  it("runs the handler with secrets from hello and reports success", async () => {
    let seenSecret: string | undefined;
    const automation = defineAutomation({ name: "x", description: "y" });
    automation.on("ticket.created", async (_payload, ctx) => {
      seenSecret = ctx.secrets.API_KEY;
      return true;
    });
    const client = new DialinClient(automation, { url: "http://instance", token: "dat_x" });
    const { ws } = await connect(client);

    ws.emit({ type: "event", run_id: "r1", event_id: "e1", topic: "ticket.created", payload: { ticket: { id: "t1" } } });
    await tick();

    expect(seenSecret).toBe("s3cr3t");
    expect(ws.sent).toContainEqual({ type: "run_started", run_id: "r1" });
    expect(ws.sent).toContainEqual({ type: "run_finished", run_id: "r1", outcome: "success" });
    client.stop();
  });

  it("reports a falsy return as a failed run, not a crash", async () => {
    const automation = defineAutomation({ name: "x", description: "y" });
    automation.on("ticket.created", async () => false);
    const client = new DialinClient(automation, { url: "http://instance", token: "dat_x" });
    const { ws } = await connect(client);

    ws.emit({ type: "event", run_id: "r1", event_id: "e1", topic: "ticket.created", payload: {} });
    await tick();

    expect(ws.sent).toContainEqual({ type: "run_finished", run_id: "r1", outcome: "failure" });
    client.stop();
  });

  it("reports a thrown error as a crash", async () => {
    const automation = defineAutomation({ name: "x", description: "y" });
    automation.on("ticket.created", async () => {
      throw new Error("boom");
    });
    const client = new DialinClient(automation, { url: "http://instance", token: "dat_x" });
    const { ws } = await connect(client);

    ws.emit({ type: "event", run_id: "r1", event_id: "e1", topic: "ticket.created", payload: {} });
    await tick();

    expect(ws.sent).toContainEqual({ type: "run_crashed", run_id: "r1", error: "boom" });
    client.stop();
  });

  it("reports a hung handler as a failed run once the timeout elapses (not a crash)", async () => {
    const automation = defineAutomation({ name: "x", description: "y" });
    automation.on("ticket.created", () => new Promise<boolean>(() => {}));
    const client = new DialinClient(automation, { url: "http://instance", token: "dat_x", timeoutMs: 20 });
    const { ws } = await connect(client);

    ws.emit({ type: "event", run_id: "r1", event_id: "e1", topic: "ticket.created", payload: {} });
    await tick(50);

    expect(ws.sent).toContainEqual({ type: "run_finished", run_id: "r1", outcome: "failure" });
    client.stop();
  });

  it("crashes a run for a topic with no registered handler", async () => {
    const automation = defineAutomation({ name: "x", description: "y" });
    const client = new DialinClient(automation, { url: "http://instance", token: "dat_x" });
    const { ws } = await connect(client);

    ws.emit({ type: "event", run_id: "r1", event_id: "e1", topic: "doc.created", payload: {} });
    await tick();

    const crashed = ws.sent.find((f) => (f as { type: string }).type === "run_crashed") as { error: string } | undefined;
    expect(crashed?.error).toMatch(/no handler registered/);
    client.stop();
  });

  it("captures plain console.log calls during the handler as run_log frames", async () => {
    const automation = defineAutomation({ name: "x", description: "y" });
    automation.on("ticket.created", async () => {
      console.log("hello from handler");
      return true;
    });
    const client = new DialinClient(automation, { url: "http://instance", token: "dat_x" });
    const { ws } = await connect(client);

    ws.emit({ type: "event", run_id: "r1", event_id: "e1", topic: "ticket.created", payload: {} });
    await tick();

    const logFrame = ws.sent.find((f) => (f as { type: string }).type === "run_log") as { log: string } | undefined;
    expect(logFrame?.log).toContain("hello from handler");
    client.stop();
  });
});
