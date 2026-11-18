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
  username?: string;
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
  // Records the source keeps archived, counted but not imported.
  archived?: number;
  people: ImportPerson[];
  statuses: ImportStatusMatch[];
  skipped: number;
  files_copied: number;
  files_linked: number;
  // Source blocks with no counterpart here, brought in as text or links.
  unsupported: number;
  relations_skipped: number;
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

// A text field a source asks for: what in it to read, or its credential.
export interface ImportField {
  label: string;
  placeholder: string;
  hint: string;
}

export interface ImportSource {
  id: string;
  label: string;
  // What it brings in, under the picker.
  hint: string;
  // What the form asks for: one of the project's repositories, or a secret with the pages, team, or link to read.
  asks: "repository" | "pages" | "team" | "link";
  // For a source that asks for a link: what in it to read, and its credential.
  from?: ImportField;
  credential?: ImportField;
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
    id: "trello",
    label: "Trello board",
    asks: "link",
    hint: "One board: its open lists as columns, and its cards with their checklists, comments, labels, files and due dates.",
    from: { label: "Board", placeholder: "https://trello.com/b/…", hint: "Its URL or short link; empty takes your only open board." },
    credential: { label: "API key and token", placeholder: "key:token", hint: "Your Trello API key and a token, joined by a colon." },
  },
  {
    id: "asana",
    label: "Asana project",
    asks: "link",
    hint: "One project: its sections as columns, and its tasks and their subtasks with comments, tags, files and due dates.",
    from: { label: "Asana project", placeholder: "https://app.asana.com/…", hint: "Its URL or id; empty takes your only project." },
    credential: { label: "Personal access token", placeholder: "Token", hint: "From the Asana developer console." },
  },
  {
    id: "notion",
    label: "Notion pages",
    hint: "The pages shared with an internal integration, as docs: one folder per root page, with their images and database rows.",
    asks: "pages",
  },
  {
    id: "linear",
    label: "Linear",
    hint: "One team: its issues with their comments, labels, priority, project, due date, assignee, files and blocking links.",
    asks: "team",
  },
];

export const importSourceLabel = (id: string): string => IMPORT_SOURCES.find((s) => s.id === id)?.label ?? id;
