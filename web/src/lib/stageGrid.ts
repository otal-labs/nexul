const GAP = 12;
const MIN_WIDTH = 160;
const MIN_HEIGHT = 100;
const MAX_LONE_WIDTH = 480;
const VIDEO = 16 / 9;

export interface StageGrid {
  columns: number;
  rows: number;
  cellWidth: number;
  cellHeight: number;
  /** Tiles drawn; when some do not fit, one more cell holds a +N tile for the rest. */
  shown: number;
  overflow: number;
}

const cell = (cells: number, columns: number, width: number, height: number) => {
  const rows = Math.ceil(cells / columns);
  return { columns, rows, cellWidth: (width - (columns - 1) * GAP) / columns, cellHeight: (height - (rows - 1) * GAP) / rows };
};

const videoWidth = (c: { cellWidth: number; cellHeight: number }) => Math.min(c.cellWidth, c.cellHeight * VIDEO);

// A cell is cut to 16:9 (never below the readable minimum), so a camera is never cropped by half and the tiles sit centred in the well.
const toVideo = <C extends { cellWidth: number; cellHeight: number }>(c: C): C => {
  const cellWidth = Math.min(c.cellWidth, c.cellHeight * VIDEO);
  return { ...c, cellWidth, cellHeight: Math.min(c.cellHeight, Math.max(MIN_HEIGHT, cellWidth / VIDEO)) };
};

// The readable column count that gives a camera feed the largest 16:9 picture, the larger cell on a tie; undefined when none is readable.
const bestFit = (cells: number, width: number, height: number) =>
  Array.from({ length: cells }, (_, i) => cell(cells, i + 1, width, height))
    .filter((c) => c.cellWidth >= MIN_WIDTH && c.cellHeight >= MIN_HEIGHT)
    .sort((a, b) => videoWidth(b) - videoWidth(a) || b.cellWidth * b.cellHeight - a.cellWidth * a.cellHeight)[0];

// Lays n call tiles over a measured well in 16:9 cells as large as it allows; one or two people sit in a row of 480px tiles.
export const stageGrid = (n: number, width: number, height: number): StageGrid => {
  if (width <= 0 || height <= 0 || n === 0) return { columns: 1, rows: 1, cellWidth: 0, cellHeight: 0, shown: n, overflow: 0 };
  if (n <= 2) {
    const columns = (width - GAP) / 2 >= MIN_WIDTH ? n : 1;
    const rows = n / columns;
    const cellWidth = Math.min(MAX_LONE_WIDTH, (width - (columns - 1) * GAP) / columns, ((height - (rows - 1) * GAP) / rows) * VIDEO);
    return { columns, rows, cellWidth, cellHeight: cellWidth / VIDEO, shown: n, overflow: 0 };
  }
  const fit = bestFit(n, width, height);
  if (fit) return { ...toVideo(fit), shown: n, overflow: 0 };
  for (let cells = n - 1; cells > 1; cells--) {
    const partial = bestFit(cells, width, height);
    if (partial) return { ...toVideo(partial), shown: cells - 1, overflow: n - cells + 1 };
  }
  return { ...cell(1, 1, width, height), shown: 0, overflow: n };
};
