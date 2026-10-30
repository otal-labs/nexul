import { create } from "zustand";

// A draft's chip lights its source's rail row; a rail row lights the questions whose drafts cite it.
export interface InterviewSourceFocus {
  sourceId: string;
  from: "chip" | "rail";
}

export type InterviewSourceFocusStore = {
  focus: InterviewSourceFocus | null;
  setFocus: (focus: InterviewSourceFocus | null) => void;
};

// The rail is sticky only beside the questions; stacked under them, whatever lights up is off screen, so nothing does.
const railBeside = () => {
  const rail = document.querySelector("[data-source-rail]");
  return !!rail && getComputedStyle(rail).position === "sticky";
};

export const useInterviewSourceFocusStore = create<InterviewSourceFocusStore>((set) => ({
  focus: null,
  setFocus: (focus) => set({ focus: focus && railBeside() ? focus : null }),
}));
