import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { THREAD_PANE_MAX, THREAD_PANE_MIN, useThreadPaneStore } from "@/stores/threadPaneStore";

const DAY_MS = 24 * 60 * 60 * 1000;
const NOW = Date.UTC(2026, 9, 3);

const savedWidths = () => JSON.parse(localStorage.getItem("thread-pane") ?? "{}").state?.widths;
const width = (ticketId: string) => useThreadPaneStore.getState().widths[ticketId]?.width;

describe("threadPaneStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useThreadPaneStore.setState({ widths: {} });
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(NOW);
  });

  afterEach(() => vi.useRealTimers());

  it("has no width for a ticket until one is chosen, so the pane keeps its share of the page", () => {
    expect(width("t1")).toBeUndefined();
  });

  it("keeps a dated width per ticket, clamped and saved to the browser, and reset clears only that ticket", () => {
    useThreadPaneStore.getState().setWidth("t1", THREAD_PANE_MIN - 100);
    useThreadPaneStore.getState().setWidth("t2", THREAD_PANE_MAX + 100);
    expect(savedWidths()).toEqual({ t1: { width: THREAD_PANE_MIN, at: NOW }, t2: { width: THREAD_PANE_MAX, at: NOW } });
    useThreadPaneStore.getState().reset("t1");
    expect(savedWidths()).toEqual({ t2: { width: THREAD_PANE_MAX, at: NOW } });
  });

  it("drops widths older than a week when another ticket is adjusted", () => {
    useThreadPaneStore.setState({
      widths: { t3: { width: 400, at: NOW - 14 * DAY_MS }, t4: { width: 420, at: NOW - 6 * DAY_MS } },
    });
    useThreadPaneStore.getState().setWidth("t5", 500);
    expect(Object.keys(useThreadPaneStore.getState().widths).sort()).toEqual(["t4", "t5"]);
  });

  it("reads saved widths back without breaking the grid on bad values", async () => {
    const stored = { widths: { t1: { width: 9000, at: NOW }, t2: { width: "wide", at: NOW }, t3: 400 } };
    localStorage.setItem("thread-pane", JSON.stringify({ state: stored, version: 0 }));
    await useThreadPaneStore.persist.rehydrate();
    expect(useThreadPaneStore.getState().widths).toEqual({ t1: { width: THREAD_PANE_MAX, at: NOW } });
  });

  it("drops the old single width shared by every ticket", async () => {
    localStorage.setItem("thread-pane", JSON.stringify({ state: { width: 400 }, version: 0 }));
    await useThreadPaneStore.persist.rehydrate();
    expect(useThreadPaneStore.getState().widths).toEqual({});
  });
});
