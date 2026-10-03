import { create } from "zustand";

import { FOLLOW_UPS, LONG_STACK_ANSWER, TEMPLATE, type ProtoQuestion } from "@/components/memory/prototype/InterviewPrototypeData";
import type { AnswerValue } from "@/models/Question";

export type ProtoRun = "idle" | "reading" | "asking" | "writing" | "ready";
export type ProtoMemory = "none" | "fresh" | "legacy";

export interface ProtoSnapshot {
  answers: Record<string, AnswerValue>;
  skipped: string[];
  openId: string | null;
  followUps: number;
  run: ProtoRun;
  memory: ProtoMemory;
  changed: string[];
  memoryDot: boolean;
  drafts: Record<string, AnswerValue>;
}

interface ProtoStore extends ProtoSnapshot {
  epoch: number;
  reset: (state: number) => void;
  open: (id: string) => void;
  back: (id: string) => void;
  save: (id: string, value: AnswerValue | null) => void;
  regenerate: () => void;
  seenMemory: () => void;
}

export const STATES = ["Empty", "Answering", "Agent reading", "Agent asking", "Memory ready", "Changing an answer", "Existing memory, no answers"];

const pick = (...selected: string[]): AnswerValue => ({ selected });

const ALL_ANSWERS: Record<string, AnswerValue> = {
  q1: { text: LONG_STACK_ANSWER },
  q2: pick("Layers (routes, logic, storage)"),
  q3: pick("Returned and wrapped"),
  q5: pick("Unit", "Integration against real dependencies", "End-to-end in this repo"),
  q6: pick("Early return, no else", "Comments only for why", "Strict types, no any"),
  q7: pick("When it saves real code"),
  q8: pick("Environment variables"),
  q9: pick("Pull request, squash merge"),
  q11: pick("Web"),
  q12: { text: "Basket: the items a shopper has picked but not paid for\nDrop: a limited release with a start time" },
};

const FOLLOW_UP_ANSWERS: Record<string, AnswerValue> = {
  f1: pick("With the change (Recommended)"),
  f2: pick("Add them here as the project grows (Recommended)"),
};

const blank: ProtoSnapshot = { answers: {}, skipped: [], openId: "q1", followUps: 0, run: "idle", memory: "none", changed: [], memoryDot: false, drafts: {} };

const done = { answers: ALL_ANSWERS, skipped: ["q4", "q10"], openId: null };

const snapshots: ProtoSnapshot[] = [
  blank,
  { ...blank, answers: { q1: ALL_ANSWERS.q1!, q2: ALL_ANSWERS.q2!, q3: ALL_ANSWERS.q3! }, skipped: ["q4"], openId: "q5", drafts: { q5: ALL_ANSWERS.q5! } },
  { ...blank, ...done, run: "reading" },
  { ...blank, ...done, run: "asking", followUps: 2, openId: "f1", drafts: { f1: FOLLOW_UP_ANSWERS.f1! } },
  { ...blank, ...done, answers: { ...ALL_ANSWERS, ...FOLLOW_UP_ANSWERS }, followUps: 2, run: "ready", memory: "fresh", memoryDot: true },
  {
    ...blank,
    ...done,
    answers: { ...ALL_ANSWERS, ...FOLLOW_UP_ANSWERS, q4: pick("Before the code") },
    skipped: ["q10"],
    openId: "q4",
    followUps: 2,
    run: "ready",
    memory: "fresh",
    changed: ["q4"],
  },
  { ...blank, run: "ready", memory: "legacy" },
];

export const rowsOf = (followUps: number): ProtoQuestion[] => [...TEMPLATE, ...FOLLOW_UPS.slice(0, followUps)];

const isDone = (s: ProtoSnapshot, id: string) => s.answers[id] !== undefined || s.skipped.includes(id);

// The next row still waiting after the one just closed, wrapping to the first one left.
const nextOpen = (s: ProtoSnapshot, after: string): string | null => {
  const rows = rowsOf(s.followUps);
  const from = rows.findIndex((q) => q.id === after);
  const ordered = [...rows.slice(from + 1), ...rows.slice(0, from)];
  return ordered.find((q) => !isDone(s, q.id))?.id ?? null;
};

// ponytail: timers stand in for the run; a reset bumps the epoch so a stale timer does nothing.
export const useInterviewPrototypeStore = create<ProtoStore>((set, get) => {
  const later = (fn: () => void) => {
    const epoch = get().epoch;
    setTimeout(() => get().epoch === epoch && fn(), 2500);
  };
  const startReading = () => {
    set({ run: "reading", openId: null });
    later(() => set({ run: "asking", followUps: 2, openId: "f1", drafts: { f1: FOLLOW_UP_ANSWERS.f1! } }));
  };
  const startWriting = () => {
    set({ run: "writing", openId: null, changed: [] });
    later(() => set({ run: "ready", memory: "fresh", memoryDot: true }));
  };

  return {
    ...blank,
    epoch: 0,
    reset: (state) => set((s) => ({ ...(snapshots[state - 1] ?? blank), epoch: s.epoch + 1 })),
    open: (id) => set({ openId: id }),
    back: (id) => {
      const rows = rowsOf(get().followUps);
      const i = rows.findIndex((q) => q.id === id);
      set({ openId: rows[Math.max(0, i - 1)]?.id ?? id });
    },
    save: (id, value) => {
      const s = get();
      const answers = { ...s.answers };
      delete answers[id];
      if (value) answers[id] = value;
      const skipped = value ? s.skipped.filter((x) => x !== id) : [...new Set([...s.skipped, id])];
      const changed = s.memory === "none" ? s.changed : [...new Set([...s.changed, id])];
      const next: ProtoSnapshot = { ...s, answers, skipped, changed };
      set({ answers, skipped, changed, openId: nextOpen(next, id) });
      const templateDone = TEMPLATE.every((q) => isDone(next, q.id));
      const followUpsDone = s.followUps > 0 && FOLLOW_UPS.every((q) => isDone(next, q.id));
      if (s.run === "idle" && s.memory === "none" && templateDone) startReading();
      if (s.run === "asking" && followUpsDone) startWriting();
    },
    regenerate: startWriting,
    seenMemory: () => set({ memoryDot: false }),
  };
});

export const VARIANTS = [
  { key: "A", label: "Side card" },
  { key: "B", label: "Memory column" },
  { key: "C", label: "Tabs" },
];
