// @vitest-environment node
import { describe, expect, it } from "vitest";

import { createFrameBatch } from "@nexul/client-core/frameBatch";

const setup = (limit = 100) => {
  const frames = new Map<number, () => void>();
  let next = 1;
  const delivered: number[][] = [];
  const batch = createFrameBatch<number>({
    deliver: (items) => delivered.push(items),
    limit,
    requestFrame: (run) => {
      frames.set(next, run);
      return next++;
    },
    cancelFrame: (handle) => frames.delete(handle),
  });
  const runFrame = () => {
    const due = [...frames.values()];
    frames.clear();
    due.forEach((run) => run());
  };
  return { batch, frames, delivered, runFrame };
};

describe("createFrameBatch", () => {
  it("delivers a burst pushed within one frame as one call, in arrival order", () => {
    const { batch, frames, delivered, runFrame } = setup();
    for (let i = 0; i < 20; i++) batch.push([i * 2, i * 2 + 1]);

    expect(frames.size).toBe(1);
    expect(delivered).toEqual([]);
    runFrame();
    expect(delivered).toEqual([Array.from({ length: 40 }, (_, i) => i)]);
  });

  it("schedules nothing for an empty push", () => {
    const { batch, frames } = setup();
    batch.push([]);
    expect(frames.size).toBe(0);
  });

  it("flush delivers what is pending now and drops the frame it had asked for", () => {
    const { batch, frames, delivered, runFrame } = setup();
    batch.push([1, 2]);
    batch.flush();

    expect(delivered).toEqual([[1, 2]]);
    expect(frames.size).toBe(0);
    runFrame();
    batch.flush();
    expect(delivered).toEqual([[1, 2]]);
  });

  it("cancel drops what is pending and its frame", () => {
    const { batch, frames, delivered } = setup();
    batch.push([1]);
    batch.cancel();

    expect(frames.size).toBe(0);
    batch.flush();
    expect(delivered).toEqual([]);
  });

  it("keeps the newest items when a frame is long in coming", () => {
    const { batch, delivered, runFrame } = setup(10);
    for (let i = 0; i < 25; i++) batch.push([i]);
    runFrame();

    const [items] = delivered;
    expect(items!.at(-1)).toBe(24);
    expect(items!.length).toBeGreaterThanOrEqual(10);
    expect(items!.length).toBeLessThanOrEqual(20);
    expect(items).toEqual([...items!].sort((a, b) => a - b));
  });

  it("asks for a new frame for what arrives after one ran", () => {
    const { batch, delivered, runFrame } = setup();
    batch.push([1]);
    runFrame();
    batch.push([2]);
    runFrame();
    expect(delivered).toEqual([[1], [2]]);
  });
});
