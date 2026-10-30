import { describe, expect, it } from "vitest";

import { stageGrid } from "@/lib/stageGrid";

describe("stageGrid", () => {
  it.each([
    // tiles, well width, well height -> columns, rows
    [1, 1100, 700, 1, 1],
    [2, 1100, 700, 2, 1],
    [4, 760, 420, 2, 2],
    [9, 760, 420, 3, 3],
    [9, 700, 330, 3, 3],
    [3, 700, 320, 2, 2],
    // the stage at 768x900 and at 1440x900 with the chat docked beside it
    [9, 636, 341, 3, 3],
    [9, 771, 739, 3, 3],
    [4, 771, 739, 2, 2],
  ])("lays %i tiles in a %ix%i well as %i columns and %i rows", (n, w, h, columns, rows) => {
    const grid = stageGrid(n, w, h);
    expect([grid.columns, grid.rows, grid.overflow]).toEqual([columns, rows, 0]);
  });

  it.each([
    [9, 760, 420],
    [4, 771, 739],
  ])("cuts %i cells in a %ix%i well to 16:9 inside it, every cell at least the minimum", (n, w, h) => {
    const grid = stageGrid(n, w, h);
    expect(grid.cellWidth / grid.cellHeight).toBeCloseTo(16 / 9);
    expect(grid.columns * grid.cellWidth + (grid.columns - 1) * 12).toBeLessThanOrEqual(w + 0.01);
    expect(grid.rows * grid.cellHeight + (grid.rows - 1) * 12).toBeLessThanOrEqual(h + 0.01);
    expect(grid.cellWidth).toBeGreaterThanOrEqual(160);
    expect(grid.cellHeight).toBeGreaterThanOrEqual(100);
  });

  it("gives the last cell to a +N tile once the tiles no longer fit", () => {
    const grid = stageGrid(20, 700, 320);
    expect(grid.cellWidth).toBeGreaterThanOrEqual(160);
    expect(grid.cellHeight).toBeGreaterThanOrEqual(100);
    expect(grid.columns * grid.rows).toBe(grid.shown + 1);
    expect(grid.shown + grid.overflow).toBe(20);
  });

  it("at 1024x768, where nine people can't all be readable, shows seven and a +2", () => {
    expect(stageGrid(9, 736, 280)).toMatchObject({ columns: 4, rows: 2, shown: 7, overflow: 2 });
  });

  it("caps one or two tiles at 480px wide", () => {
    expect(stageGrid(1, 1100, 700).cellWidth).toBe(480);
    expect(stageGrid(2, 1100, 700).cellWidth).toBe(480);
  });

  it("returns an empty grid for a well not measured yet", () => {
    expect(stageGrid(3, 0, 0)).toMatchObject({ shown: 3, overflow: 0 });
  });
});
