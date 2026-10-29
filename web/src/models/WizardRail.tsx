// Every wizard's rail draws a step in one of these: where you are, what the run recorded, or a step its groundwork still locks.
export type WizardRailMark = "current" | "done" | "skipped" | "later" | "left" | "locked";

export const RAIL_MARK_WORD: Record<WizardRailMark, string> = {
  current: "Current",
  done: "Done",
  skipped: "Skipped",
  later: "Later",
  left: "To do",
  locked: "Locked",
};
