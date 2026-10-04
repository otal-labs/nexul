import { create } from "zustand";

import {
  ANSWERS,
  DRAFT_ORDER,
  FOLLOW_UPS,
  QUESTIONS,
  SOURCES_LATER,
  SOURCES_SUPERSEDING,
  SOURCES_THREE,
  SUGGESTIONS,
  type ProtoDraft,
  type ProtoSource,
  type SourceKind,
  type Stance,
} from "@/components/memory/prototype/SourcesProtoData";
import type { AnswerValue, QuestionItem } from "@/models/Question";

export type DraftRun = "idle" | "drafting" | "ready";
export type RowState = "answered" | "skipped" | "drafted" | "pending";

interface Snapshot {
  sources: ProtoSource[];
  adding: SourceKind | null;
  run: DraftRun;
  drafted: string[];
  answers: Record<string, AnswerValue>;
  skipped: string[];
  followUps: number;
  suggestions: Record<string, ProtoDraft>;
  openId: string | null;
  memory: boolean;
  changed: number;
  sourcesChanged: boolean;
  folds: Record<string, boolean>;
}

interface Store extends Snapshot {
  epoch: number;
  reset: (state: number) => void;
  open: (id: string | null) => void;
  save: (id: string, value: AnswerValue | null) => void;
  accept: (id: string) => void;
  dismiss: (id: string) => void;
  setAdding: (kind: SourceKind | null) => void;
  addSource: (source: Omit<ProtoSource, "id">) => void;
  removeSource: (id: string) => void;
  setStance: (id: string, stance: Stance) => void;
  draft: () => void;
  toggleFold: (key: string, folded: boolean) => void;
}

export const STATES = ["No sources", "Adding a source", "Drafting", "Drafts ready", "A source added later", "Superseding project"];

export const VARIANTS = [
  { key: "A", label: "Sources section" },
  { key: "B", label: "Memory column" },
  { key: "C", label: "Page strip" },
];

const only = (...ids: string[]) => Object.fromEntries(ids.map((id) => [id, ANSWERS[id]!]));

const blank: Snapshot = {
  sources: [], adding: null, run: "idle", drafted: [], answers: {}, skipped: [], followUps: 0,
  suggestions: {}, openId: "q1", memory: false, changed: 0, sourcesChanged: false, folds: {},
};

const snapshots: Snapshot[] = [
  blank,
  { ...blank, adding: "text" },
  { ...blank, sources: SOURCES_THREE, run: "drafting", drafted: DRAFT_ORDER.slice(0, 3) },
  { ...blank, sources: SOURCES_THREE, run: "ready", drafted: DRAFT_ORDER, answers: only("q2", "q4"), openId: "q5" },
  {
    ...blank, sources: SOURCES_LATER, run: "ready", drafted: DRAFT_ORDER, answers: { ...ANSWERS }, followUps: 2,
    suggestions: { ...SUGGESTIONS }, openId: "q4", memory: true, sourcesChanged: true,
  },
  { ...blank, sources: SOURCES_SUPERSEDING },
];

export const rowsOf = (followUps: number): QuestionItem[] => [...QUESTIONS, ...FOLLOW_UPS.slice(0, followUps)];

export const rowState = (s: Pick<Snapshot, "answers" | "skipped" | "drafted">, id: string): RowState => {
  if (s.answers[id]) return "answered";
  if (s.skipped.includes(id)) return "skipped";
  if (s.drafted.includes(id)) return "drafted";
  return "pending";
};

const waiting = (s: Snapshot, id: string) => ["drafted", "pending"].includes(rowState(s, id));

const nextOpen = (s: Snapshot, after: string): string | null => {
  const rows = rowsOf(s.followUps);
  const from = rows.findIndex((r) => r.id === after);
  return [...rows.slice(from + 1), ...rows.slice(0, from)].find((r) => waiting(s, r.id))?.id ?? null;
};

// ponytail: timers stand in for the drafting run; a reset bumps the epoch so a stale timer does nothing.
export const useSourcesProtoStore = create<Store>((set, get) => {
  const later = (ms: number, fn: () => void) => {
    const epoch = get().epoch;
    setTimeout(() => get().epoch === epoch && fn(), ms);
  };
  const tick = () =>
    later(1800, () => {
      const s = get();
      const next = DRAFT_ORDER.find((id) => !s.drafted.includes(id) && !s.answers[id]);
      if (!next) {
        set({ run: "ready", sourcesChanged: false });
        return;
      }
      set({ drafted: [...s.drafted, next] });
      tick();
    });

  return {
    ...blank,
    epoch: 0,
    reset: (state) => {
      set((s) => ({ ...(snapshots[state - 1] ?? blank), epoch: s.epoch + 1 }));
      if (state === 3) tick();
    },
    open: (id) => set({ openId: id }),
    save: (id, value) => {
      const s = get();
      const answers = { ...s.answers };
      delete answers[id];
      if (value) answers[id] = value;
      const skipped = value ? s.skipped.filter((x) => x !== id) : [...s.skipped, id];
      const next = { ...s, answers, skipped };
      set({ answers, skipped, openId: nextOpen(next, id), changed: s.memory ? s.changed + 1 : 0 });
    },
    accept: (id) => {
      const s = get();
      const suggestions = { ...s.suggestions };
      const value = suggestions[id]?.value;
      delete suggestions[id];
      set({ suggestions, answers: { ...s.answers, ...(value && { [id]: value }) }, changed: s.changed + 1, openId: null });
    },
    dismiss: (id) => {
      const suggestions = { ...get().suggestions };
      delete suggestions[id];
      set({ suggestions, openId: null });
    },
    setAdding: (adding) => set({ adding }),
    addSource: (source) =>
      set((s) => ({ sources: [...s.sources, { ...source, id: `n${s.sources.length}`, fresh: true }], adding: null, sourcesChanged: true })),
    removeSource: (id) => set((s) => ({ sources: s.sources.filter((x) => x.id !== id) })),
    setStance: (id, stance) =>
      set((s) => ({ sources: s.sources.map((x) => (x.id === id ? { ...x, stance } : x)), sourcesChanged: true })),
    draft: () => {
      const s = get();
      set({ run: "drafting", drafted: s.drafted.filter((id) => s.answers[id]) });
      tick();
    },
    toggleFold: (key, folded) => set((s) => ({ folds: { ...s.folds, [key]: folded } })),
  };
});

export const followCount = (sources: ProtoSource[]) => sources.filter((x) => x.stance === "follow").length;

export const sourcesMeta = (sources: ProtoSource[]): string => {
  const follow = followCount(sources);
  const noun = sources.length === 1 ? "source" : "sources";
  return `${sources.length} ${noun} · ${follow} follow · ${sources.length - follow} question`;
};
