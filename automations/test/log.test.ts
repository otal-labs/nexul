import { afterEach, describe, expect, jest, spyOn, test } from "bun:test";
import { LogShipper } from "../src/log-shipper.ts";
import { log, logScript, LogQueue, MAX_BATCH, QUEUE_LIMIT, shipLogsTo, type LogRecord } from "../src/log.ts";

function rec(message: string, extra: Partial<LogRecord> = {}): LogRecord {
  return { time: "2026-10-01T12:00:00.000Z", level: "info", message, ...extra };
}

describe("LogQueue", () => {
  test("keeps the newest QUEUE_LIMIT records and says how many it dropped, once", () => {
    const q = new LogQueue();
    for (let i = 0; i < QUEUE_LIMIT + 5; i++) q.add(rec(`line ${i}`));

    expect(q.length).toBe(QUEUE_LIMIT);
    const first = q.peek(3);
    expect(first.dropped).toBe(5);
    expect(first.records.map((r) => r.message)).toEqual(["dropped 5 log lines", "line 5", "line 6"]);
    expect(first.records[0]?.level).toBe("warn");

    q.ack(first.next, first.dropped);
    const second = q.peek(1);
    expect(second.dropped).toBe(0);
    expect(second.records[0]?.message).toBe("line 7");
    expect(q.length).toBe(QUEUE_LIMIT - 2);
  });

  test("a batch that was never acknowledged is still there, drop count included", () => {
    const q = new LogQueue();
    for (let i = 0; i < QUEUE_LIMIT + 2; i++) q.add(rec(`line ${i}`));

    const a = q.peek();
    const b = q.peek();
    expect(b.records.map((r) => r.message)).toEqual(a.records.map((r) => r.message));
    expect(b.dropped).toBe(2);
    expect(q.length).toBe(QUEUE_LIMIT);
  });

  test("overflow during a send only costs what it dropped", () => {
    const q = new LogQueue();
    for (let i = 0; i < QUEUE_LIMIT; i++) q.add(rec(`line ${i}`));
    const sent = q.peek(10);
    for (let i = 0; i < 4; i++) q.add(rec(`late ${i}`));
    q.ack(sent.next, sent.dropped);

    const rest = q.peek(1);
    expect(rest.dropped).toBe(4);
    expect(rest.records[0]?.message).toBe("dropped 4 log lines");
    expect(q.length).toBe(QUEUE_LIMIT - 6);
  });

  test("a batch holds at most MAX_BATCH records and fits its byte budget", () => {
    const q = new LogQueue();
    for (let i = 0; i < 200; i++) q.add(rec("x"));
    expect(q.peek().records).toHaveLength(MAX_BATCH);

    const big = new LogQueue();
    for (let i = 0; i < 5; i++) big.add(rec("y".repeat(1500)));
    expect(big.peek(MAX_BATCH, 4000).records).toHaveLength(2);
    expect(big.peek(MAX_BATCH, 10).records).toHaveLength(1);
  });

  test("redacts tokens and cuts long text before anything leaves", () => {
    const token = `ses_${"a".repeat(50)}`;
    const q = new LogQueue();
    q.add(rec(`failed with ${token}`, { attrs: { url: "wss://example.com/x?token=abc", nested: { auth: token }, long: "z".repeat(3000), n: 3 } }));

    const [r] = q.peek().records;
    expect(r?.message).toBe("failed with [redacted token]");
    expect(r?.attrs?.url).toBe("wss://example.com/x?token=[redacted token]");
    expect(r?.attrs?.nested).toEqual({ auth: "[redacted token]" });
    expect((r?.attrs?.long as string).length).toBe(2001);
    expect(r?.attrs?.n).toBe(3);
  });

  test("a record too big to send keeps its message and loses its attributes", () => {
    const q = new LogQueue();
    const attrs = Object.fromEntries(Array.from({ length: 40 }, (_, i) => [`k${i}`, "v".repeat(1900)]));
    q.add(rec("huge", { attrs }));

    const [r] = q.peek().records;
    expect(r?.message).toBe("huge");
    expect(r?.attrs).toBeUndefined();
  });
});

describe("log and logScript", () => {
  afterEach(() => shipLogsTo(null));

  test("log queues the host's own line, a script's line stays local", () => {
    const out = spyOn(console, "log").mockImplementation(() => {});
    const err = spyOn(console, "error").mockImplementation(() => {});
    const q = new LogQueue();
    shipLogsTo(q);

    log("info", "automations host starting", { name: "jobs" });
    log("error", "poll tick failed");
    logScript("info", "hello from a script", { automationId: "a1" });

    expect(q.peek().records.map((r) => [r.level, r.message, r.attrs])).toEqual([
      ["info", "automations host starting", { name: "jobs" }],
      ["error", "poll tick failed", undefined],
    ]);
    expect(out).toHaveBeenCalledTimes(2);
    expect(err).toHaveBeenCalledTimes(1);
    out.mockRestore();
    err.mockRestore();
  });

  test("nothing is queued until shipping is on", () => {
    const out = spyOn(console, "log").mockImplementation(() => {});
    log("info", "before shipping");
    out.mockRestore();
    const q = new LogQueue();
    shipLogsTo(q);
    expect(q.length).toBe(0);
  });
});

describe("LogShipper", () => {
  test("sends the queue in batches and empties it", async () => {
    const q = new LogQueue();
    for (let i = 0; i < 120; i++) q.add(rec(`line ${i}`));
    const batches: number[] = [];
    const shipper = new LogShipper(q, async (records) => {
      batches.push(records.length);
    });

    await shipper.flush();

    expect(batches).toEqual([50, 50, 20]);
    expect(q.length).toBe(0);
  });

  test("a failed post leaves everything queued for the next flush, and nothing is thrown", async () => {
    const q = new LogQueue();
    q.add(rec("keep me"));
    let up = false;
    const seen: string[][] = [];
    const shipper = new LogShipper(q, async (records) => {
      if (!up) throw new Error("connection refused");
      seen.push(records.map((r) => r.message));
    });

    await shipper.flush();
    expect(q.length).toBe(1);

    up = true;
    await shipper.flush();
    expect(seen).toEqual([["keep me"]]);
    expect(q.length).toBe(0);
  });

  test("a flush that is already running is joined, not doubled", async () => {
    const q = new LogQueue();
    q.add(rec("once"));
    let calls = 0;
    let release: () => void = () => {};
    const shipper = new LogShipper(q, () => {
      calls++;
      return new Promise<void>((resolve) => {
        release = resolve;
      });
    });

    const a = shipper.flush();
    const b = shipper.flush();
    release();
    await Promise.all([a, b]);

    expect(calls).toBe(1);
  });

  test("start flushes on every interval until stopped", async () => {
    jest.useFakeTimers();
    try {
      const q = new LogQueue();
      const posted: string[][] = [];
      const shipper = new LogShipper(q, async (records) => {
        posted.push(records.map((r) => r.message));
      });

      shipper.start(5);
      q.add(rec("first"));
      jest.advanceTimersByTime(5);
      expect(posted).toEqual([["first"]]);
      await shipper.flush();
      q.add(rec("second"));
      jest.advanceTimersByTime(5);
      expect(posted).toEqual([["first"], ["second"]]);
      await shipper.flush();

      shipper.stop();
      q.add(rec("after stop"));
      jest.advanceTimersByTime(1000);
      expect(posted).toHaveLength(2);
      expect(q.length).toBe(1);
    } finally {
      jest.useRealTimers();
    }
  });
});
