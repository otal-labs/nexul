export const ImportState = {
  Scanning: "scanning",
  Ready: "ready",
  Importing: "importing",
  Completed: "completed",
  Failed: "failed",
  Cancelled: "cancelled",
} as const;

export type ImportState = (typeof ImportState)[keyof typeof ImportState];

// Someone the source named and the member login they map to; an empty member is nobody here.
export interface ImportPerson {
  login?: string;
  email?: string;
  name: string;
  member: string;
}

// A source status and the column it lands in; new is a column the import adds.
export interface ImportStatusMatch {
  name: string;
  kind: string;
  column: string;
  new: boolean;
}

export interface ImportFailure {
  kind: string;
  source_id: string;
  title?: string;
  error: string;
}

export interface ImportReport {
  tickets: number;
  comments: number;
  docs: number;
  people: ImportPerson[];
  statuses: ImportStatusMatch[];
  skipped: number;
  files_copied: number;
  files_linked: number;
  // Source blocks with no counterpart here, brought in as text or links.
  unsupported: number;
  failures: ImportFailure[];
  error?: string;
}

export interface Import {
  id: string;
  workspace_id: string;
  project_id: string;
  source: string;
  // What in the source is imported, such as owner/name for a GitHub repository.
  from: string;
  state: ImportState;
  created_by: string;
  total: number;
  done: number;
  failed: number;
  report: ImportReport;
  created_at: string;
  updated_at: string;
}

export const importOpen = (imp: Import): boolean =>
  imp.state === ImportState.Scanning || imp.state === ImportState.Ready || imp.state === ImportState.Importing;

export interface ImportSource {
  id: string;
  label: string;
  // What it brings in, under the picker.
  hint: string;
  // What the form asks for: one of the project's repositories, or an integration secret and the pages to start from.
  asks: "repository" | "pages";
}

// The sources the server reads; each new source is one entry here and one on the server.
export const IMPORT_SOURCES: ImportSource[] = [
  {
    id: "github",
    label: "GitHub issues",
    hint: "One repository attached to the project: its issues, comments, labels and first assignee.",
    asks: "repository",
  },
  {
    id: "notion",
    label: "Notion pages",
    hint: "The pages shared with an internal integration, as docs: one folder per root page, with their images and database rows.",
    asks: "pages",
  },
];

export const importSourceLabel = (id: string): string => IMPORT_SOURCES.find((s) => s.id === id)?.label ?? id;
