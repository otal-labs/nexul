import type { ActionGate } from "@nexul/client-core/actionState";

// Mirrors the history routes every versioned entity answers (internal/docs/model.go VersionSummary and DocVersion).
export interface HistoryVersionSummary {
  version: number;
  created_at: string;
  /** A named version's name; empty for the rest. */
  name: string;
  editor_ids: string[];
  /** The automation's name, "mcp" for an agent's save, empty for the browser. */
  via: string;
  /** True on the newest version. */
  current: boolean;
}

export interface History {
  versions: HistoryVersionSummary[];
}

export interface HistoryVersion {
  version: number;
  title: string;
  /** The body as the entity stores it: a Tiptap JSON string, or markdown on an old record. */
  body: string;
}

// What restoring a version takes: who may do it, and the entity's own request.
export interface HistoryRestore extends Pick<ActionGate, "permission" | "projectId" | "answer"> {
  run: (version: number) => void;
  pending: boolean;
}
