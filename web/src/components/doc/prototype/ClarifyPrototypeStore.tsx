import { useSearchParams } from "react-router";
import { create } from "zustand";

import { NOTE_1, NOTE_2, ROUND_1_ANSWERS, ROUND_2_ANSWERS, ROUND_3_ANSWERS, ROUNDS, type ProtoQuestion } from "@/components/doc/prototype/ClarifyPrototypeData";
import type { AnswerValue } from "@/models/Question";

export type ProtoPhase = "open" | "running" | "nogaps" | "closed";

export interface ProtoSnapshot {
  rounds: number;
  answers: Record<string, AnswerValue>;
  skipped: string[];
  notes: Record<number, string>;
  openId: string | null;
  phase: ProtoPhase;
  written: boolean;
  reopened: boolean;
  folds: Record<number, boolean>;
  tab: "doc" | "questions";
  blockOpen: boolean;
}

interface ProtoStore extends ProtoSnapshot {
  epoch: number;
  reset: (state: number) => void;
  open: (id: string | null) => void;
  back: (id: string) => void;
  save: (id: string, value: AnswerValue | null) => void;
  setNote: (round: number, text: string) => void;
  clarify: () => void;
  close: () => void;
  toggleFold: (round: number, folded: boolean) => void;
  setTab: (tab: "doc" | "questions") => void;
  setBlockOpen: (open: boolean) => void;
}

export const STATES = ["Round 1 arrived", "Mid round 1", "Round 2 running", "Round 1 answered", "Round 3 open", "No gaps left", "Closed"];

export const VARIANTS = [
  { key: "A", label: "Side column" },
  { key: "B", label: "Above the body" },
  { key: "C", label: "Tab" },
];

const blank: ProtoSnapshot = {
  rounds: 1, answers: {}, skipped: [], notes: {}, openId: "r1q1", phase: "open",
  written: false, reopened: false, folds: {}, tab: "questions", blockOpen: true,
};

const round1Done = { answers: ROUND_1_ANSWERS, skipped: ["r1q2"], notes: { 1: NOTE_1 }, openId: null };
const allDone = {
  rounds: 3,
  answers: { ...ROUND_1_ANSWERS, ...ROUND_2_ANSWERS, ...ROUND_3_ANSWERS },
  skipped: ["r1q2", "r2q4", "r3q1"],
  notes: { 1: NOTE_1, 2: NOTE_2 },
  openId: null,
  written: true,
};

const snapshots: ProtoSnapshot[] = [
  blank,
  { ...blank, answers: { r1q1: ROUND_1_ANSWERS.r1q1!, r1q3: ROUND_1_ANSWERS.r1q3! }, skipped: ["r1q2"], openId: "r1q4" },
  { ...blank, ...round1Done, phase: "running" },
  { ...blank, ...round1Done },
  { ...blank, rounds: 3, answers: { ...ROUND_1_ANSWERS, ...ROUND_2_ANSWERS }, skipped: ["r1q2", "r2q4"], notes: { 1: NOTE_1, 2: NOTE_2 }, openId: "r3q1" },
  { ...blank, ...allDone, phase: "nogaps" },
  { ...blank, ...allDone, phase: "closed" },
];

export const roundQuestions = (n: number): ProtoQuestion[] => ROUNDS[n - 1]?.questions ?? [];

export const questionsUpTo = (rounds: number): ProtoQuestion[] => ROUNDS.slice(0, rounds).flatMap((r) => r.questions);

export const roundOf = (id: string): number => Number(id.slice(1, id.indexOf("q")));

export const isDone = (s: Pick<ProtoSnapshot, "answers" | "skipped">, id: string): boolean =>
  s.answers[id] !== undefined || s.skipped.includes(id);

export const pendingCount = (s: ProtoSnapshot): number =>
  s.phase === "open" ? questionsUpTo(s.rounds).filter((q) => !isDone(s, q.id)).length : 0;

export const waitingLabel = (n: number): string => `${n} ${n === 1 ? "question" : "questions"} waiting`;

// The next question still waiting after the one just closed, wrapping round to the first one left.
const nextOpen = (s: ProtoSnapshot, after: string): string | null => {
  const rows = questionsUpTo(s.rounds);
  const from = rows.findIndex((q) => q.id === after);
  const ordered = [...rows.slice(from + 1), ...rows.slice(0, Math.max(from, 0))];
  return ordered.find((q) => !isDone(s, q.id))?.id ?? null;
};

const withViews = (s: ProtoSnapshot): ProtoSnapshot => {
  const waiting = pendingCount(s) > 0;
  return { ...s, tab: waiting ? "questions" : "doc", blockOpen: waiting };
};

// ponytail: a timer stands in for the run; a reset bumps the epoch so a stale timer does nothing.
export const useClarifyPrototypeStore = create<ProtoStore>((set, get) => {
  const arrive = () => {
    const s = get();
    if (s.rounds < 3 || (s.reopened && s.rounds < 4)) {
      const n = s.rounds + 1;
      set({ rounds: n, phase: "open", openId: roundQuestions(n)[0]?.id ?? null, folds: {}, tab: "questions", blockOpen: true });
      return;
    }
    set({ phase: "nogaps", written: true, openId: null });
  };

  return {
    ...blank,
    epoch: 0,
    reset: (state) => set((s) => ({ ...withViews(snapshots[state - 1] ?? blank), epoch: s.epoch + 1 })),
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
    clarify: () => {
      const s = get();
      const epoch = s.epoch;
      set({ phase: "running", openId: null, reopened: s.reopened || s.phase === "closed" });
      setTimeout(() => get().epoch === epoch && arrive(), 2500);
    },
    close: () => set({ phase: "closed", openId: null }),
    toggleFold: (round, folded) => set((s) => ({ folds: { ...s.folds, [round]: !folded } })),
    setTab: (tab) => set({ tab }),
    setBlockOpen: (blockOpen) => set({ blockOpen }),
  };
});

// ?as=dev shows what someone who can run plays sees; anything else is the client's view.
export const useClarifyDev = (): boolean => useSearchParams()[0].get("as") === "dev";
