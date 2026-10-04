import { create } from "zustand";

import {
  NOTE_1,
  NOTE_2,
  ROUND_1_ANSWERS,
  ROUND_2_ANSWERS,
  ROUND_3_ANSWERS,
  ROUNDS,
  type AnswerValue,
  type ProtoQuestion,
} from "@/components/docs/prototype/ClarifyProtoData";

export type ProtoPhase = "open" | "running" | "closed";
export type ProtoVariant = "A" | "B" | "C";
export type ProtoTab = "doc" | "questions";
// What B's full-screen view shows: one question, the round's "Anything else?" page, or every round.
export type ProtoStep = { kind: "question"; id: string } | { kind: "note" } | { kind: "overview" };

interface ProtoSnapshot {
  rounds: number;
  answers: Record<string, AnswerValue>;
  skipped: string[];
  notes: Record<number, string>;
  openId: string | null;
  phase: ProtoPhase;
  folds: Record<number, boolean>;
  tab: ProtoTab;
  sheetOpen: boolean;
  step: ProtoStep | null;
}

interface ProtoStore extends ProtoSnapshot {
  variant: ProtoVariant;
  reset: (state: number, variant: ProtoVariant) => void;
  open: (id: string | null) => void;
  back: (id: string) => void;
  save: (id: string, value: AnswerValue | null) => void;
  setNote: (round: number, text: string) => void;
  toggleFold: (round: number, folded: boolean) => void;
  setTab: (tab: ProtoTab) => void;
  setSheetOpen: (open: boolean) => void;
  setStep: (step: ProtoStep | null) => void;
}

export const STATES = [
  "From the inbox",
  "Round arrived",
  "Mid round",
  "Round being written",
  "All answered",
  "Round 3, earlier folded",
  "Closed",
];

export const VARIANTS: { key: ProtoVariant; label: string }[] = [
  { key: "A", label: "Tab" },
  { key: "B", label: "One per screen" },
  { key: "C", label: "Bottom sheet" },
];

const blank: ProtoSnapshot = {
  rounds: 1, answers: {}, skipped: [], notes: {}, openId: "r1q1", phase: "open",
  folds: {}, tab: "questions", sheetOpen: false, step: null,
};

const round1Done = { answers: ROUND_1_ANSWERS, skipped: ["r1q2"], notes: { 1: NOTE_1 }, openId: null };

// Index 0 is the inbox, so the doc behind it is the round that just arrived.
const snapshots: ProtoSnapshot[] = [
  blank,
  blank,
  { ...blank, answers: { r1q1: ROUND_1_ANSWERS.r1q1!, r1q3: ROUND_1_ANSWERS.r1q3! }, skipped: ["r1q2"], openId: "r1q4" },
  { ...blank, ...round1Done, phase: "running" },
  { ...blank, ...round1Done },
  { ...blank, rounds: 3, answers: { ...ROUND_1_ANSWERS, ...ROUND_2_ANSWERS }, skipped: ["r1q2", "r2q4"], notes: { 1: NOTE_1, 2: NOTE_2 }, openId: "r3q1" },
  {
    ...blank,
    rounds: 3,
    answers: { ...ROUND_1_ANSWERS, ...ROUND_2_ANSWERS, ...ROUND_3_ANSWERS },
    skipped: ["r1q2", "r2q4", "r3q1"],
    notes: { 1: NOTE_1, 2: NOTE_2 },
    openId: null,
    phase: "closed",
  },
];

export const roundQuestions = (n: number): ProtoQuestion[] => ROUNDS[n - 1]?.questions ?? [];

export const questionsUpTo = (rounds: number): ProtoQuestion[] => ROUNDS.slice(0, rounds).flatMap((r) => r.questions);

export const roundOf = (id: string): number => Number(id.slice(1, id.indexOf("q")));

export const isDone = (s: Pick<ProtoSnapshot, "answers" | "skipped">, id: string): boolean =>
  s.answers[id] !== undefined || s.skipped.includes(id);

export const pendingCount = (s: Pick<ProtoSnapshot, "answers" | "skipped" | "phase" | "rounds">): number =>
  s.phase === "open" ? questionsUpTo(s.rounds).filter((q) => !isDone(s, q.id)).length : 0;

export const waitingLabel = (n: number): string => `${n} ${n === 1 ? "question" : "questions"} waiting`;

// The next question still waiting after the one just closed, wrapping round to the first one left.
const nextOpen = (s: ProtoSnapshot, after: string): string | null => {
  const rows = questionsUpTo(s.rounds);
  const from = rows.findIndex((q) => q.id === after);
  const ordered = [...rows.slice(from + 1), ...rows.slice(0, Math.max(from, 0))];
  return ordered.find((q) => !isDone(s, q.id))?.id ?? null;
};

export const useClarifyProtoStore = create<ProtoStore>((set, get) => ({
  ...blank,
  variant: "A",
  reset: (state, variant) => {
    const snap = snapshots[state] ?? blank;
    set({ ...snap, variant, tab: pendingCount(snap) > 0 ? "questions" : "doc", folds: {}, sheetOpen: false, step: null });
  },
  open: (id) => set((s) => ({ openId: id, folds: id ? { ...s.folds, [roundOf(id)]: false } : s.folds })),
  back: (id) => {
    const rows = questionsUpTo(get().rounds);
    const prev = rows[Math.max(0, rows.findIndex((q) => q.id === id) - 1)]?.id ?? id;
    get().open(prev);
  },
  save: (id, value) => {
    const s = get();
    const answers = { ...s.answers };
    delete answers[id];
    if (value) answers[id] = value;
    const skipped = value ? s.skipped.filter((x) => x !== id) : [...new Set([...s.skipped, id])];
    set({ answers, skipped, openId: nextOpen({ ...s, answers, skipped }, id) });
  },
  setNote: (round, text) => set((s) => ({ notes: { ...s.notes, [round]: text } })),
  toggleFold: (round, folded) => set((s) => ({ folds: { ...s.folds, [round]: !folded } })),
  setTab: (tab) => set({ tab }),
  setSheetOpen: (sheetOpen) => set({ sheetOpen }),
  setStep: (step) => set({ step }),
}));
