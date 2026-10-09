import { z } from "zod";

import type { AnswerValue } from "@/models/Question";
import type { Trail } from "@/models/Trail";

// Mirrors internal/memories SourcePath..SourceText.
export const SOURCE_KINDS = ["path", "doc", "memory", "project", "text"] as const;
export type SourceKind = (typeof SOURCE_KINDS)[number];

export const SOURCE_STANCES = ["follow", "question"] as const;
export type SourceStance = (typeof SOURCE_STANCES)[number];

export const SOURCE_KIND_LABEL: Record<SourceKind, string> = {
  path: "Path",
  doc: "Doc",
  memory: "Memory",
  project: "Project",
  text: "Paste text",
};

export const SOURCE_STANCE_LABEL: Record<SourceStance, string> = { follow: "Follow", question: "Question" };

// Mirrors memories.InterviewSource; label, not_visible, gone, and ref_updated_at are resolved for the reader.
export interface InterviewSource {
  id: string;
  workspace_id: string;
  project_id: string;
  kind: SourceKind;
  ref: string;
  label: string;
  body?: string;
  stance: SourceStance;
  ref_updated_at?: string;
  not_visible?: boolean;
  gone?: boolean;
  added_by: string;
  added_at: string;
  updated_at: string;
}

// Mirrors memories.InterviewDraft: a proposed answer to a template question, never an answer itself.
export interface InterviewDraft {
  id: string;
  workspace_id: string;
  project_id: string;
  question: string;
  selected: string[];
  text: string;
  source_ids: string[];
  where: string;
  trail_id?: string;
  drafted_by: string;
  drafted_at: string;
}

export interface AddInterviewSourceInput {
  project_id: string;
  kind: SourceKind;
  ref: string;
  label: string;
  body: string;
  stance: SourceStance;
}

// Mirrors memories.MaxSourceTextChars.
export const MAX_SOURCE_TEXT_CHARS = 32_000;

export const AddSourceFormSchema = z
  .object({
    kind: z.enum(SOURCE_KINDS),
    path: z.string(),
    ref: z.string(),
    label: z.string(),
    body: z.string().max(MAX_SOURCE_TEXT_CHARS, `Paste at most ${MAX_SOURCE_TEXT_CHARS.toLocaleString()} characters`),
    // "" until the person picks one; the kind's default stands in for it.
    stance: z.enum(SOURCE_STANCES).or(z.literal("")),
  })
  .superRefine((v, ctx) => {
    if (v.kind === "path" && v.path.trim() === "") ctx.addIssue({ code: "custom", path: ["path"], message: "Enter a path" });
    if (v.kind === "text" && v.label.trim() === "") ctx.addIssue({ code: "custom", path: ["label"], message: "Give the text a label" });
    if (v.kind === "text" && v.body.trim() === "") ctx.addIssue({ code: "custom", path: ["body"], message: "Paste some text" });
    const picked = v.kind === "doc" || v.kind === "memory" || v.kind === "project";
    if (picked && v.ref === "") ctx.addIssue({ code: "custom", path: ["ref"], message: `Choose a ${SOURCE_KIND_LABEL[v.kind].toLowerCase()}` });
  });

export type AddSourceFormData = z.infer<typeof AddSourceFormSchema>;

// What the add form preselects: follow for a doc, memory, text, markdown file, or folder (trailing "/"); question otherwise.
export const defaultStance = (kind: SourceKind, path: string): SourceStance => {
  if (kind === "project") return "question";
  if (kind !== "path") return "follow";
  const p = path.trim().toLowerCase();
  return p.endsWith(".md") || p.endsWith(".markdown") || p.endsWith("/") ? "follow" : "question";
};

export const formStance = (v: Pick<AddSourceFormData, "kind" | "path" | "stance">): SourceStance =>
  v.stance === "" ? defaultStance(v.kind, v.path) : v.stance;

export const toAddSourceInput = (projectId: string, v: AddSourceFormData): AddInterviewSourceInput => {
  const ref: Record<SourceKind, string> = { path: v.path.trim(), doc: v.ref, memory: v.ref, project: v.ref, text: "" };
  const text = v.kind === "text";
  return { project_id: projectId, kind: v.kind, ref: ref[v.kind], label: text ? v.label.trim() : "", body: text ? v.body : "", stance: formStance(v) };
};

// A file the browser can read as pasted text; anything else is refused.
export const isReadableTextFile = (file: File): boolean =>
  file.type.startsWith("text/") || /\.(md|markdown|txt)$/i.test(file.name);

export const isNamed = (source: InterviewSource): boolean => !source.gone && !source.not_visible;

// What to call a source: its name, or only its kind when it is gone or hidden from this reader.
export const sourceName = (source: InterviewSource): string =>
  isNamed(source) ? source.label : SOURCE_KIND_LABEL[source.kind];

export const sourcesMeta = (sources: InterviewSource[]): string => {
  if (sources.length === 0) return "None yet";
  const follow = sources.filter((s) => s.stance === "follow").length;
  const noun = sources.length === 1 ? "source" : "sources";
  return `${sources.length} ${noun} · ${follow} follow · ${sources.length - follow} question`;
};

// When the last finished drafting run started; trails come newest first.
export const lastDraftedAt = (trails: Trail[]): number | undefined => {
  const last = trails.find((t) => t.state === "done");
  return last && Date.parse(last.started_at);
};

const after = (at: string | undefined, since: number): boolean => at !== undefined && Date.parse(at) > since;

// New when added after the drafting run started; changed when its doc, memory, or text was. Paths and projects can't tell.
export const sourceChange = (s: InterviewSource, since: number | undefined): "new" | "changed" | null => {
  if (since === undefined) return null;
  if (after(s.added_at, since)) return "new";
  if ((s.kind === "doc" || s.kind === "memory") && after(s.ref_updated_at, since)) return "changed";
  if (s.kind === "text" && after(s.updated_at, since)) return "changed";
  return null;
};

export const sourcesChangedSince = (sources: InterviewSource[], trails: Trail[]): boolean => {
  const since = lastDraftedAt(trails);
  return sources.some((s) => sourceChange(s, since) !== null);
};

export const draftValue = (draft: InterviewDraft): AnswerValue => ({ selected: draft.selected, text: draft.text });
