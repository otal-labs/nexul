import { z } from "zod";

// An interview play runs on a project's Interview page; its run target id is the project id.
export const PLAY_TYPES = ["ticket", "doc", "interview"] as const;
export type PlayType = (typeof PLAY_TYPES)[number];

export const PLAY_TYPE_LABELS: Record<PlayType, string> = {
  ticket: "Ticket",
  doc: "Doc",
  interview: "Interview",
};

// Mirrors the Go backend's five fixed board stages (ADR 0022); a ticket play shows on exactly one.
export const PLAY_STAGES = ["backlog", "progress", "review", "testing", "done"] as const;
export type PlayStage = (typeof PLAY_STAGES)[number];

export const PLAY_STAGE_LABELS: Record<PlayStage, string> = {
  backlog: "Backlog",
  progress: "In progress",
  review: "Review",
  testing: "Testing",
  done: "Done",
};

export interface Play {
  id: string;
  workspace_id: string;
  label: string;
  type: PlayType;
  description: string;
  instructions: string;
  enabled: boolean;
  show_when_stage: PlayStage | null;
  excluded_project_ids: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

// show_when_stage is "" for no stage (any play but a ticket play) rather than null, so it plugs straight into FormSelect's
// empty-value sentinel; the refine below is the client-side half of the same rule the API enforces.
export const SavePlayFormSchema = z
  .object({
    label: z.string().min(1, "Label is required"),
    type: z.enum(PLAY_TYPES),
    description: z.string(),
    instructions: z.string(),
    enabled: z.boolean(),
    show_when_stage: z.string(),
    excluded_project_ids: z.array(z.string()),
  })
  .refine(
    (data) =>
      data.type === "ticket"
        ? (PLAY_STAGES as readonly string[]).includes(data.show_when_stage)
        : data.show_when_stage === "",
    {
      message: "A ticket play needs exactly one show-when stage; other plays can't have one",
      path: ["show_when_stage"],
    },
  );

export type SavePlayFormData = z.infer<typeof SavePlayFormSchema>;

// The wire shape the gateway accepts: show_when_stage travels as null, never "".
export interface SavePlayRequest {
  label: string;
  type: PlayType;
  description: string;
  instructions: string;
  enabled: boolean;
  show_when_stage: PlayStage | null;
  excluded_project_ids: string[];
}

export const toSavePlayRequest = (data: SavePlayFormData): SavePlayRequest => ({
  ...data,
  show_when_stage: data.show_when_stage === "" ? null : (data.show_when_stage as PlayStage),
});
