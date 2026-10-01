import { beforeEach, describe, expect, it } from "vitest";

import { LIST_PANE_DEFAULT, LIST_PANE_MAX, LIST_PANE_MIN, useListPaneStore } from "@/stores/listPaneStore";

describe("listPaneStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useListPaneStore.getState().reset();
  });

  it("starts at the default width", () => {
    expect(useListPaneStore.getState().width).toBe(LIST_PANE_DEFAULT);
  });

  it("clamps a width below the minimum and above the maximum", () => {
    useListPaneStore.getState().setWidth(LIST_PANE_MIN - 100);
    expect(useListPaneStore.getState().width).toBe(LIST_PANE_MIN);
    useListPaneStore.getState().setWidth(LIST_PANE_MAX + 100);
    expect(useListPaneStore.getState().width).toBe(LIST_PANE_MAX);
  });

  it("persists the chosen width to the browser and reset puts the default back", () => {
    useListPaneStore.getState().setWidth(400);
    expect(JSON.parse(localStorage.getItem("list-pane") ?? "{}").state.width).toBe(400);
    useListPaneStore.getState().reset();
    expect(useListPaneStore.getState().width).toBe(LIST_PANE_DEFAULT);
  });

  it("clamps an out-of-range width read back from the browser", async () => {
    localStorage.setItem("list-pane", JSON.stringify({ state: { width: 9000 }, version: 0 }));
    await useListPaneStore.persist.rehydrate();
    expect(useListPaneStore.getState().width).toBe(LIST_PANE_MAX);
  });
});
