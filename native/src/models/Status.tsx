// The fixed, ordered board stages a status column belongs to; mirrors internal/workspace's Status.Kind.
export const StatusKind = {
  Backlog: "backlog",
  Progress: "progress",
  Review: "review",
  Testing: "testing",
  Done: "done",
} as const;

export type StatusKind = (typeof StatusKind)[keyof typeof StatusKind];

// The board's stages in order with their hues, the same table as the web's.
export const STATUS_STAGES: { kind: StatusKind; label: string; dot: string }[] = [
  { kind: StatusKind.Backlog, label: "Backlog", dot: "bg-muted-foreground" },
  { kind: StatusKind.Progress, label: "Progress", dot: "bg-info" },
  { kind: StatusKind.Review, label: "Review", dot: "bg-warning" },
  { kind: StatusKind.Testing, label: "Testing", dot: "bg-foreground" },
  { kind: StatusKind.Done, label: "Done", dot: "bg-success" },
];

export const statusStageDot = (kind: StatusKind): string =>
  (STATUS_STAGES.find((stage) => stage.kind === kind) ?? STATUS_STAGES[0]!).dot;

// A project's status column; the server already returns these ordered by kind then position (the board's column order).
export interface BoardStatus {
  id: string;
  name: string;
  position: number;
  kind: StatusKind;
}
