import { z } from "zod";

import type { TrailState } from "@/models/Trail";

// An interview play runs on a project's Interview page; its run target id is the project id. A stack play runs on a stack.
export const PLAY_TYPES = ["ticket", "doc", "interview", "stack"] as const;
export type PlayType = (typeof PLAY_TYPES)[number];

export const PLAY_TYPE_LABELS: Record<PlayType, string> = {
  ticket: "Ticket",
  doc: "Doc",
  interview: "Interview",
  stack: "Stack",
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

// The built-in interview play that drafts answers from sources; it is never the interview's run.
export const DRAFT_INTERVIEW_KEY = "interview-draft";

// The built-in interview play that audits code against the interview memory and writes a doc; it is never the interview's run.
export const AUDIT_KEY = "audit";

// The built-in ticket play a ticket entering done starts through its seeded auto play.
export const DECISIONS_CHECK_KEY = "decisions-check";

// The built-in stack play a crashed service starts through its seeded auto play, once someone picks its computer.
export const INVESTIGATE_CRASH_KEY = "investigate-crash";

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
  // How many of its auto plays are switched on; the list sends it only to someone who reads auto plays.
  auto_plays_on?: number;
  // The seeded play this one is (fix-with-ai, to-tickets-via-ai, interview, test-with-ai, interview-draft, clarify, update-doc, audit, decisions-check, investigate-crash), kept through renames; "" for a play a person made.
  builtin_key: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// A play as a list hands it to one person: can_run is whether its own rules and their roles let them run it.
export interface ListedPlay extends Play {
  can_run: boolean;
}

// A play on the workspace's plays list, which alone carries what its runs say about it.
export interface WorkspacePlay extends ListedPlay, PlayRunHealth {}

export const needAnswers = (n: number) => (n === 1 ? "1 needs an answer" : `${n} need answers`);

// What needs a look now, busiest first: runs in flight, then runs waiting on an answer, then a newest run that failed.
export const playAttention = (play: WorkspacePlay): { state: TrailState; text: string } | undefined => {
  if (play.running > 0) return { state: "running", text: `${play.running} running` };
  if (play.waiting > 0) return { state: "waiting", text: needAnswers(play.waiting) };
  if (play.last_run?.state === "failed") return { state: "failed", text: "Last run failed" };
  return undefined;
};

// Mirrors internal/plays.RunHealth: what the runs this person can see say about a play. last_run is null until it has run.
export interface PlayRunHealth {
  last_run: { state: TrailState; started_at: string } | null;
  runs_7d: number;
  failed_7d: number;
  waiting: number;
  running: number;
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
      message: "A ticket play needs one show-when stage. Other plays have none.",
      path: ["show_when_stage"],
    },
  );

export type SavePlayFormData = z.infer<typeof SavePlayFormSchema>;

export const emptyPlayForm = (): SavePlayFormData => ({
  label: "",
  type: "ticket",
  description: "",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
});

export const playToFormData = (play: Play): SavePlayFormData => ({
  label: play.label,
  type: play.type,
  description: play.description,
  instructions: play.instructions,
  enabled: play.enabled,
  show_when_stage: play.show_when_stage ?? "",
  excluded_project_ids: play.excluded_project_ids,
});

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

// The automations whose code still calls a play by its current label with runPlay; names only for someone who reads automations.
export interface PlayRenameImpact {
  count: number;
  automations: string[];
}
