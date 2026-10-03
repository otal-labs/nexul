import { beforeEach, describe, expect, it } from "vitest";

import { THREAD_PANE_MAX, THREAD_PANE_MIN, useThreadPaneStore } from "@/stores/threadPaneStore";

const savedWidths = () => JSON.parse(localStorage.getItem("thread-pane") ?? "{}").state?.widths;
const width = (ticketId: string) => useThreadPaneStore.getState().widths[ticketId];

describe("threadPaneStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useThreadPaneStore.setState({ widths: {} });
  });

  it("has no width for a ticket until one is chosen, so the pane keeps its share of the page", () => {
    expect(width("t1")).toBeUndefined();
  });

  it("keeps a width per ticket, clamped and saved to the browser, and reset clears only that ticket", () => {
    useThreadPaneStore.getState().setWidth("t1", THREAD_PANE_MIN - 100);
    useThreadPaneStore.getState().setWidth("t2", THREAD_PANE_MAX + 100);
    expect(savedWidths()).toEqual({ t1: THREAD_PANE_MIN, t2: THREAD_PANE_MAX });
    useThreadPaneStore.getState().reset("t1");
    expect(savedWidths()).toEqual({ t2: THREAD_PANE_MAX });
  });

  it("reads saved widths back without breaking the grid on bad values", async () => {
    const stored = { widths: { t1: 9000, t2: "wide" } };
    localStorage.setItem("thread-pane", JSON.stringify({ state: stored, version: 0 }));
    await useThreadPaneStore.persist.rehydrate();
    expect(useThreadPaneStore.getState().widths).toEqual({ t1: THREAD_PANE_MAX });
  });

  it("drops the old single width shared by every ticket", async () => {
    localStorage.setItem("thread-pane", JSON.stringify({ state: { width: 400 }, version: 0 }));
    await useThreadPaneStore.persist.rehydrate();
    expect(useThreadPaneStore.getState().widths).toEqual({});
  });
});
