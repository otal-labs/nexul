// The fixed, ordered board stages a status column belongs to; mirrors internal/workspace's Status.Kind.
export const StatusKind = {
  Backlog: "backlog",
  Progress: "progress",
  Review: "review",
  Testing: "testing",
  Done: "done",
} as const;

export type StatusKind = (typeof StatusKind)[keyof typeof StatusKind];

const STAGE_DOT: Record<StatusKind, string> = {
  [StatusKind.Backlog]: "bg-muted-foreground",
  [StatusKind.Progress]: "bg-info",
  [StatusKind.Review]: "bg-warning",
  [StatusKind.Testing]: "bg-foreground",
  [StatusKind.Done]: "bg-success",
};

export const statusStageDot = (kind: StatusKind): string => STAGE_DOT[kind] ?? STAGE_DOT[StatusKind.Backlog];

// A project's status column; the server already returns these ordered by kind then position (the board's column order).
export interface BoardStatus {
  id: string;
  name: string;
  position: number;
  kind: StatusKind;
}
