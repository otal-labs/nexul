import type { BoardStatus } from "@/models/Status";

// Every lane's row and the stage header share this template, so a stage is one column down the whole board.
// A stage grows to 280px and no further: past it a wide screen would widen every card, not the work.
export const boardGridTemplate = (columns: BoardStatus[], doneOpen: boolean): string =>
  columns.map((column) => (column.kind === "done" && !doneOpen ? "7.5rem" : "minmax(11.25rem, 17.5rem)")).join(" ");

export const isFolded = (column: BoardStatus, doneOpen: boolean) => column.kind === "done" && !doneOpen;

const tracks = (columns: BoardStatus[], doneOpen: boolean) => {
  const folded = columns.filter((column) => isFolded(column, doneOpen)).length;
  return { open: columns.length - folded, folded };
};

// The widest the board ever gets: every open stage at 280px, each folded one at 7.5rem, and the gaps.
export const boardGridMaxWidth = (columns: BoardStatus[], doneOpen: boolean): string => {
  const { open, folded } = tracks(columns, doneOpen);
  return `calc(${open} * 17.5rem + ${folded} * 7.5rem + ${Math.max(columns.length - 1, 0)} * 0.5rem)`;
};

// The narrowest the board may be before it scrolls sideways: every open stage at its minimum, folded Done, and the gaps.
export const boardGridMinWidth = (columns: BoardStatus[], doneOpen: boolean): string => {
  const { open, folded } = tracks(columns, doneOpen);
  return `calc(${open} * 11.25rem + ${folded} * 7.5rem + ${Math.max(columns.length - 1, 0)} * 0.5rem)`;
};

// The tickets on show per status: the stage row's counts and the header's bar both read from this.
export const countByStatus = (lanes: { tickets: { status: string }[] }[]): Record<string, number> => {
  const counts: Record<string, number> = {};
  for (const ticket of lanes.flatMap((lane) => lane.tickets)) counts[ticket.status] = (counts[ticket.status] ?? 0) + 1;
  return counts;
};

// Done sorts last, so folded Done is the board's trailing track: a lane band stops before it, where no well is drawn.
export const foldedTrailWidth = (columns: BoardStatus[], doneOpen: boolean): string | undefined => {
  const { folded } = tracks(columns, doneOpen);
  return folded > 0 ? `${folded * 8}rem` : undefined;
};
