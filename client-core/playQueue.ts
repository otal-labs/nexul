// What a queued run's wording reads: its status, the server's hold reason, and the computers it waits on (internal/plays/queue.go).
export interface WaitingRun {
  status: string;
  reason: string;
  target_type: string;
  waiting_on?: { why: string; text: string }[];
}

// Why none of the person's computers can take the run: a few words for a short line, a sentence for a longer one (internal/pairing NotConfiguredError).
export const SETUP_GAPS: Record<string, { why: string; sentence: string }> = {
  unpaired: { why: "no computer paired", sentence: "no computer is paired to run it on" },
  expired_token: { why: "computer session expired", sentence: "the paired computer's session expired" },
  no_computer_ready: { why: "no computer can take it", sentence: "none of their computers can take it" },
  offline: { why: "computer offline", sentence: "their computer is offline" },
  setup_required: { why: "computer setup not done", sentence: "the computer's setup isn't done" },
};

const waitingOn = (item: WaitingRun) => item.waiting_on ?? [];

// Every computer that couldn't take the run was only busy, or left out by its auto play's computer rule, so it starts once a busy one frees up.
export const onlyComputersFull = (item: WaitingRun) =>
  waitingOn(item).some((p) => p.why === "full") && waitingOn(item).every((p) => p.why === "full" || p.why === "rule_out");

// Why a queued run waits, from the server's hold reasons.
export const waitingWhy = (item: WaitingRun): string => {
  if (item.status === "dispatching") return "starting";
  if (item.reason === "ticket busy") return `${item.target_type} busy`;
  if (onlyComputersFull(item)) return "computers busy";
  return SETUP_GAPS[item.reason]?.why ?? "next in line";
};

// waitingWhy as sentences: one per computer and why it can't take the run when the server named them, else the gap.
export const waitingLines = (item: WaitingRun): string[] => {
  if (item.status !== "queued") return [waitingWhy(item)];
  if (waitingOn(item).length > 0) return waitingOn(item).map((p) => p.text);
  return [SETUP_GAPS[item.reason]?.sentence || waitingWhy(item)];
};
