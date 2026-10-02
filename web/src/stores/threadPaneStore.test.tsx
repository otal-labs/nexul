import { beforeEach, describe, expect, it } from "vitest";

import { THREAD_PANE_MAX, THREAD_PANE_MIN, useThreadPaneStore } from "@/stores/threadPaneStore";

const savedWidth = () => JSON.parse(localStorage.getItem("thread-pane") ?? "{}").state?.width;

describe("threadPaneStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useThreadPaneStore.getState().reset();
  });

  it("has no width until one is chosen, so the pane keeps its share of the page", () => {
    expect(useThreadPaneStore.getState().width).toBeNull();
  });

  it("clamps a chosen width, saves it to the browser, and reset goes back to no width", () => {
    useThreadPaneStore.getState().setWidth(THREAD_PANE_MIN - 100);
    expect(useThreadPaneStore.getState().width).toBe(THREAD_PANE_MIN);
    useThreadPaneStore.getState().setWidth(THREAD_PANE_MAX + 100);
    expect(savedWidth()).toBe(THREAD_PANE_MAX);
    useThreadPaneStore.getState().reset();
    expect(useThreadPaneStore.getState().width).toBeNull();
    expect(savedWidth()).toBeNull();
  });

  it.each([
    ["an out-of-range number", 9000, THREAD_PANE_MAX],
    ["a value that is not a number", "wide", null],
  ])("reads %s back from the browser without breaking the grid", async (_name, stored, expected) => {
    localStorage.setItem("thread-pane", JSON.stringify({ state: { width: stored }, version: 0 }));
    await useThreadPaneStore.persist.rehydrate();
    expect(useThreadPaneStore.getState().width).toBe(expected);
  });
});
