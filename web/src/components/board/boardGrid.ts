import type { BoardStatus } from "@/models/Status";

// Every lane's row and the stage header share this template, so a stage is one column down the whole board.
export const boardGridTemplate = (columns: BoardStatus[], doneOpen: boolean): string =>
  columns.map((column) => (column.kind === "done" && !doneOpen ? "2.75rem" : "minmax(11.25rem, 1fr)")).join(" ");

export const isFolded = (column: BoardStatus, doneOpen: boolean) => column.kind === "done" && !doneOpen;
