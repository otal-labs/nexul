import { DEFAULT_ERROR_RATE, type ErrorRateRule, type Stack, type StackWithBranches } from "@/models/Stack";
import { formatShare } from "@/utils/RequestsUtility";

export const errorRateOf = (stack: Pick<Stack, "error_rate">): ErrorRateRule => stack.error_rate ?? DEFAULT_ERROR_RATE;

const minutes = (n: number): string => `${n} minute${n === 1 ? "" : "s"}`;

// "5xx above 5% for 5 minutes, with at least 20 requests"
export const ruleText = (rule: ErrorRateRule): string =>
  `5xx above ${formatShare(rule.percent / 100)} for ${minutes(rule.minutes)}, with at least ${rule.min_requests.toLocaleString()} requests`;

export const recoveryText = (rule: ErrorRateRule): string =>
  `Recovers after ${minutes(rule.minutes)} at or under ${formatShare(rule.percent / 100)}.`;

export interface ErrorRateHealth {
  tone: "warning" | "success";
  text: string;
  // The time the episode began, or ended for a recovery.
  at: string;
}

// The Health column's line for the rule: the open episode in the warning hue, a recovery within the hour in success.
// Nothing while the rule is off, since an episode it ended by being switched off is not a recovery.
export const errorRateHealth = (stack: StackWithBranches): ErrorRateHealth | undefined => {
  if (!errorRateOf(stack).enabled) return undefined;
  const open = stack.error_rate_episode;
  if (open) return { tone: "warning", text: `5xx ${formatShare(open.peak_percent / 100)}`, at: open.started_at };
  if (stack.error_rate_recovered_at) return { tone: "success", text: "5xx recovered", at: stack.error_rate_recovered_at };
  return undefined;
};
