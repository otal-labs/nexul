import type { BoardStatus } from "@/models/Status";

// Every lane's row and the stage header share this template, so a stage is one column down the whole board.
export const boardGridTemplate = (columns: BoardStatus[], doneOpen: boolean): string =>
  columns.map((column) => (column.kind === "done" && !doneOpen ? "2.75rem" : "minmax(11.25rem, 1fr)")).join(" ");

export const isFolded = (column: BoardStatus, doneOpen: boolean) => column.kind === "done" && !doneOpen;

// The narrowest the board may be before it scrolls sideways: every open stage at its minimum, folded Done, and the gaps.
export const boardGridMinWidth = (columns: BoardStatus[], doneOpen: boolean): string => {
  const folded = columns.filter((column) => isFolded(column, doneOpen)).length;
  return `calc(${columns.length - folded} * 11.25rem + ${folded} * 2.75rem + ${Math.max(columns.length - 1, 0)} * 0.5rem)`;
};
