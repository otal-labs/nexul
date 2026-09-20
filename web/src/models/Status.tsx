import { z } from "zod";

// The fixed, ordered board stages a status column belongs to; a project skips one by having no columns in it.
export const StatusKind = {
  Backlog: "backlog",
  Progress: "progress",
  Review: "review",
  Testing: "testing",
  Done: "done",
} as const;

export type StatusKind = (typeof StatusKind)[keyof typeof StatusKind];

export const STATUS_STAGES: { kind: StatusKind; label: string; text: string; dot: string }[] = [
  { kind: StatusKind.Backlog, label: "Backlog", text: "text-muted-foreground", dot: "bg-muted-foreground" },
  { kind: StatusKind.Progress, label: "Progress", text: "text-info", dot: "bg-info" },
  { kind: StatusKind.Review, label: "Review", text: "text-warning", dot: "bg-warning" },
  { kind: StatusKind.Testing, label: "Testing", text: "text-foreground", dot: "bg-foreground" },
  { kind: StatusKind.Done, label: "Done", text: "text-success", dot: "bg-success" },
];

const fallbackStage = STATUS_STAGES[0]!;

export const statusStage = (kind: StatusKind) => STATUS_STAGES.find((stage) => stage.kind === kind) ?? fallbackStage;

// Values are lucide-react component names stored verbatim, looked up directly by StatusIcon.tsx.
export const STATUS_ICON_NAMES = [
  "CircleDashed",
  "Circle",
  "CircleDot",
  "CircleEllipsis",
  "CircleCheckBig",
  "CircleX",
] as const;

export type StatusIconName = (typeof STATUS_ICON_NAMES)[number];

export interface BoardStatus {
  id: string;
  name: string;
  position: number;
  kind: StatusKind;
  icon: string;
  created_at: string;
  updated_at: string;
}

export const SaveStatusFormSchema = z.object({
  name: z.string().trim().min(1, "Status name is required"),
  kind: z.enum([StatusKind.Backlog, StatusKind.Progress, StatusKind.Review, StatusKind.Testing, StatusKind.Done]),
  icon: z.enum(["", ...STATUS_ICON_NAMES]),
});

export type SaveStatusFormData = z.infer<typeof SaveStatusFormSchema>;
