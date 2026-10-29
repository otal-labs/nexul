import type { HistoryFieldChange, HistoryVersionSummary } from "@/models/History";
import { formatVersionTime } from "@/utils/TimeUtility";

// The server keeps this many unnamed versions per item (internal/docs/model.go KeptUnnamedVersions).
export const KEPT_UNNAMED_VERSIONS = 10;

// "mcp" is an agent's save; anything else names the automation it came through.
export const viaLabel = (via: string) => (via === "mcp" ? "via an agent" : `via ${via}`);

// Three digits at most: "999", then "1.2k", "12k".
export const cappedCount = (n: number): string => {
  if (n < 1000) return String(n);
  if (n < 10_000) return `${(n / 1000).toFixed(1)}k`;
  return `${Math.round(n / 1000)}k`;
};

export interface VersionSize {
  added: number;
  removed: number;
  /** The exact figures, for the row's accessible name and its tooltip. */
  exact: string;
}

// The size the server stored with the version; none on the oldest kept, on a server that sends none, or when no word changed, so the slot stays empty.
export const versionSize = (summary: HistoryVersionSummary): VersionSize | undefined => {
  const { words_added: added, words_removed: removed } = summary;
  if (typeof added !== "number" || typeof removed !== "number" || added + removed === 0) return undefined;
  return { added, removed, exact: `${added} words added, ${removed} removed` };
};

// What a restore copied: the source's time while it is still kept, else its number once a prune has dropped it; empty on a version that is not a restore.
export const restoredWhen = (summary: HistoryVersionSummary, versions: HistoryVersionSummary[]): string => {
  if (typeof summary.restored_from !== "number") return "";
  const source = versions.find((v) => v.version === summary.restored_from);
  return source ? formatVersionTime(source.created_at) : `version ${summary.restored_from}`;
};

export const restoredLabel = (summary: HistoryVersionSummary, versions: HistoryVersionSummary[]): string => {
  const when = restoredWhen(summary, versions);
  return when && `Restored from ${when}`;
};

export interface KeptCounts {
  /** "12 versions · 2 named": everything kept, then the named ones among them. */
  short: string;
  /** The rule behind the cap, for the count's tooltip and accessible name. */
  long: string;
}

// The rule says "plus named ones" only where a named version can exist: the item can name versions, or one is already kept.
export const keepsNamed = (versions: HistoryVersionSummary[], namesVersions: boolean): boolean => namesVersions || versions.some((v) => v.name);

export const keptCounts = (versions: HistoryVersionSummary[], namesVersions = true): KeptCounts => {
  const named = versions.filter((v) => v.name).length;
  const keeps = keepsNamed(versions, namesVersions);
  return {
    short: `${versions.length} ${versions.length === 1 ? "version" : "versions"}${named > 0 ? ` · ${named} named` : ""}`,
    long: keeps ? `Only the last ${KEPT_UNNAMED_VERSIONS} unnamed versions are kept, plus named ones.` : `Only the last ${KEPT_UNNAMED_VERSIONS} versions are kept.`,
  };
};

export type RailItem = { kind: "version"; summary: HistoryVersionSummary } | { kind: "field"; change: HistoryFieldChange };

const timeOf = (item: RailItem) => Date.parse(item.kind === "version" ? item.summary.created_at : item.change.created_at);

// Versions and field changes in one list, newest first; a field change falls between the versions it happened among.
export const railItems = (versions: HistoryVersionSummary[], changes: HistoryFieldChange[] = []): RailItem[] =>
  [
    ...versions.map((summary): RailItem => ({ kind: "version", summary })),
    ...changes.map((change): RailItem => ({ kind: "field", change })),
  ].sort((a, b) => timeOf(b) - timeOf(a));
