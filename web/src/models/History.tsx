import type { ActionGate } from "@nexul/client-core/actionState";

// Mirrors the history routes every versioned entity answers (internal/docs/model.go VersionSummary and DocVersion).
export interface HistoryVersionSummary {
  version: number;
  created_at: string;
  /** The title a memory had at this version; only memories send it. */
  title?: string;
  /** A named version's name; empty for the rest. */
  name: string;
  editor_ids: string[];
  /** The automation's name, "mcp" for an agent's save, empty for the browser. */
  via: string;
  /** True on the newest version. */
  current: boolean;
  /** The seeded text of a built-in play nobody has edited; it stands where a named version's headline does. */
  builtin_default?: boolean;
  /** Words added and removed against the kept version before it; null on the oldest kept. */
  words_added?: number | null;
  words_removed?: number | null;
  /** The version number a restore copied this version from; null on any other version. */
  restored_from?: number | null;
  /** An automation's version id and place in its code history, which merge and rollback act on; other items send neither. */
  id?: string;
  status?: "active" | "pending" | "inactive";
}

// Who made a change and through what: "user:mcp" is a person's agent, "play" a play run, "automation" an automation.
export interface HistoryActor {
  kind?: string;
  user_id?: string;
  automation_name?: string;
  play_label?: string;
}

// One field a save changed, as a ticket's or play's history lists it: from and to are display-ready, empty for nothing.
export interface HistoryFieldChange {
  id: string;
  field: string;
  from: string;
  to: string;
  created_at: string;
  actor: HistoryActor;
}

export interface History {
  versions: HistoryVersionSummary[];
  changes?: HistoryFieldChange[];
}

export interface HistoryVersion {
  version: number;
  /** Absent on an item with no title of its own, such as a ticket's description. */
  title?: string;
  /** The body as the entity stores it: a Tiptap JSON string, or markdown on an old record. */
  body: string;
}

// How a version's body is stored, so the dialog knows how to cut it into blocks: Tiptap JSON (docs, tickets), markdown text (play instructions), or code, one block per line.
export type HistoryBodyKind = "tiptap" | "markdown" | "code";

// What an item keeps: the last ten unnamed versions plus named ones (docs), the last ten (items that can't name versions), or all of them.
export type HistoryRetention = "ten-plus-named" | "ten" | "all";

export type HistoryView = "changes" | "as-it-was";

// What restoring a version takes: who may do it, and the entity's own request.
export interface HistoryRestore extends Pick<ActionGate, "permission" | "projectId" | "answer"> {
  run: (version: number) => void;
  pending: boolean;
}
