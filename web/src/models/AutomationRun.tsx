import { CheckCircle2, Skull, XCircle, type LucideIcon } from "lucide-react";

import { AutomationRunOutcome } from "@/enums/Automation";

// Mirrors internal/automations/runs.go's Run: one delivery attempt's execution report and logs.
export interface AutomationRun {
  id: string;
  automation_id: string;
  event_topic: string;
  event_id: string;
  outcome: AutomationRunOutcome;
  error?: string;
  started_at: string;
  finished_at: string;
  duration_ms: number;
  logs: string;
  created_at: string;
}

// One mark per outcome, read by the run rows, the run sheet and the list's status mark.
export const runOutcomes: Record<AutomationRunOutcome, { icon: LucideIcon; color: string; word: string; past: string }> = {
  [AutomationRunOutcome.Success]: { icon: CheckCircle2, color: "text-success", word: "Success", past: "succeeded" },
  [AutomationRunOutcome.Failure]: { icon: XCircle, color: "text-destructive", word: "Failure", past: "failed" },
  [AutomationRunOutcome.Crash]: { icon: Skull, color: "text-destructive", word: "Crash", past: "crashed" },
};
